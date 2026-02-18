package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/ryanbastic/go-mezzanine/internal/api"
	"github.com/ryanbastic/go-mezzanine/internal/config"
	"github.com/ryanbastic/go-mezzanine/internal/index"
	"github.com/ryanbastic/go-mezzanine/internal/metrics"
	"github.com/ryanbastic/go-mezzanine/internal/shard"
	"github.com/ryanbastic/go-mezzanine/internal/storage"
	"github.com/ryanbastic/go-mezzanine/internal/trigger"
)

func main() {
	cfg := config.Load()

	var logLevel slog.Level
	switch cfg.LogLevel {
	case "debug":
		logLevel = slog.LevelDebug
	case "warn":
		logLevel = slog.LevelWarn
	case "error":
		logLevel = slog.LevelError
	case "info":
		logLevel = slog.LevelInfo
	default:
		logLevel = slog.LevelInfo
		slog.Warn("invalid log level, defaulting to info", "value", cfg.LogLevel)
	}

	const modulePrefix = "github.com/ryanbastic/go-mezzanine/"
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level:     logLevel,
		AddSource: true,
		ReplaceAttr: func(_ []string, a slog.Attr) slog.Attr {
			if a.Key == slog.SourceKey {
				if src, ok := a.Value.Any().(*slog.Source); ok {
					if idx := strings.Index(src.File, modulePrefix); idx != -1 {
						src.File = src.File[idx+len(modulePrefix):]
					}
					return slog.Attr{
						Key: a.Key,
						Value: slog.GroupValue(
							slog.String("f", src.File),
							slog.Int("l", src.Line),
							slog.String("c", src.Function),
						),
					}
				}
			}
			return a
		},
	}))
	slog.SetDefault(logger)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Load shard config
	shardCfg, err := config.LoadShardConfig(cfg.ShardConfigPath, cfg.NumShards)
	if err != nil {
		logger.Error("failed to load shard config", "error", err)
		os.Exit(1)
	}

	// Create one pool per backend, ping each
	pools := make(map[string]*pgxpool.Pool, len(shardCfg.Backends))
	for _, b := range shardCfg.Backends {
		poolCfg, err := pgxpool.ParseConfig(b.DatabaseURL)
		if err != nil {
			logger.Error("failed to parse database URL", "backend", b.Name, "error", err)
			os.Exit(1)
		}
		poolCfg.MaxConns = int32(cfg.DBMaxConns)
		poolCfg.MinConns = int32(cfg.DBMinConns)
		poolCfg.MaxConnLifetime = cfg.DBMaxConnLifetime
		poolCfg.MaxConnIdleTime = cfg.DBMaxConnIdleTime
		poolCfg.HealthCheckPeriod = cfg.DBHealthCheckPeriod

		pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
		if err != nil {
			logger.Error("failed to connect to backend", "backend", b.Name, "error", err)
			os.Exit(1)
		}
		if err := pool.Ping(ctx); err != nil {
			logger.Error("failed to ping backend", "backend", b.Name, "error", err)
			os.Exit(1)
		}
		pools[b.Name] = pool
		logger.Info("connected to backend", "backend", b.Name, "shards", []int{b.ShardStart, b.ShardEnd},
			"maxConns", cfg.DBMaxConns, "minConns", cfg.DBMinConns)
	}
	defer func() {
		for name, pool := range pools {
			pool.Close()
			logger.Info("closed pool", "backend", name)
		}
	}()

	// Register pgxpool metrics collector
	prometheus.MustRegister(metrics.NewPoolCollector(pools))
	logger.Info("registered pool metrics collector")

	logger.Info("running migrations")
	// Run migrations per backend
	for _, b := range shardCfg.Backends {
		logger.Info("running migrations for backend", "backend", b.Name, "shards", []int{b.ShardStart, b.ShardEnd})
		pool := pools[b.Name]
		if err := storage.RunMigrationsForPool(ctx, pool, b.ShardStart, b.ShardEnd); err != nil {
			logger.Error("failed to run migrations", "backend", b.Name, "error", err)
			os.Exit(1)
		}
		logger.Info("migrations complete", "backend", b.Name, "shards", []int{b.ShardStart, b.ShardEnd})
	}

	// Build shard-to-pool mapping and register stores
	router := shard.NewRouter()

	for _, b := range shardCfg.Backends {
		pool := pools[b.Name]
		for i := b.ShardStart; i <= b.ShardEnd; i++ {
			s := storage.NewPostgresStore(pool, i, cfg.DBQueryTimeout)
			router.Register(shard.ID(i), s)
		}
	}

	// Initialize index registry
	indexRegistry := index.NewRegistry()
	indexRegistry.SetQueryTimeout(cfg.DBQueryTimeout)

	if cfg.IndexConfigPath != "" {
		logger.Info("loading index config", "path", cfg.IndexConfigPath)
		idxCfg, err := config.LoadIndexConfig(cfg.IndexConfigPath)
		if err != nil {
			logger.Error("failed to load index config", "error", err)
			os.Exit(1)
		}
		logger.Info("index config loaded", "indexCount", len(idxCfg.Indexes))

		logger.Info("registering indexes")
		// Register all definitions across all backends
		for _, b := range shardCfg.Backends {
			pool := pools[b.Name]
			for _, idx := range idxCfg.Indexes {
				indexRegistry.RegisterRange(pool, index.Definition{
					Name:          idx.Name,
					SourceColumn:  idx.SourceColumn,
					ShardKeyField: idx.ShardKeyField,
					Fields:        idx.Fields,
					UniqueFields:  idx.UniqueFields,
				}, b.ShardStart, b.ShardEnd)
			}
		}

		// Create index tables per backend
		for _, b := range shardCfg.Backends {
			logger.Info("creating index tables", "backend", b.Name, "shards", []int{b.ShardStart, b.ShardEnd})
			pool := pools[b.Name]
			if err := indexRegistry.CreateTablesRange(ctx, pool, b.ShardStart, b.ShardEnd); err != nil {
				logger.Error("failed to create index tables", "backend", b.Name, "error", err)
				os.Exit(1)
			}
			logger.Info("index tables created", "backend", b.Name, "shards", []int{b.ShardStart, b.ShardEnd})
		}

		logger.Info("indexes registered", "count", len(idxCfg.Indexes))
	}

	// Initialize trigger plugin system with persistent storage.
	// Use the first backend's pool for the shared plugins table.
	firstBackend := shardCfg.Backends[0]
	pluginPool := pools[firstBackend.Name]
	if err := storage.RunPluginMigration(ctx, pluginPool); err != nil {
		logger.Error("failed to run plugin migration", "error", err)
		os.Exit(1)
	}
	pluginStore := trigger.NewPostgresPluginStore(pluginPool, cfg.DBQueryTimeout)
	pluginRegistry := trigger.NewPluginRegistry(pluginStore)
	if err := pluginRegistry.LoadAll(ctx); err != nil {
		logger.Error("failed to load plugins from store", "error", err)
		os.Exit(1)
	}
	logger.Info("plugin registry loaded", "count", len(pluginRegistry.List()))
	rpcClient := trigger.NewRPCClient(cfg.TriggerRetryMax, cfg.TriggerRetryBackoff, cfg.TriggerRPCTimeout)
	notifier := trigger.NewNotifier(pluginRegistry, rpcClient, logger)

	// Build backend pinger map for readiness checks
	backends := make(map[string]api.Pinger, len(pools))
	for name, pool := range pools {
		backends[name] = pool
	}

	// Start HTTP server
	handler := api.NewServer(logger, router, indexRegistry, pluginRegistry, notifier, cfg.NumShards, backends)
	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      handler,
		ReadTimeout:  cfg.HTTPReadTimeout,
		WriteTimeout: cfg.HTTPWriteTimeout,
		IdleTimeout:  cfg.HTTPIdleTimeout,
	}

	go func() {
		logger.Info("starting HTTP server", "port", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("HTTP server error", "error", err)
			os.Exit(1)
		}
	}()

	// Graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh
	logger.Info("shutting down...")

	// Cancel context to stop trigger watchers
	cancel()

	// Graceful HTTP shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error("HTTP shutdown error", "error", err)
	}

	logger.Info("shutdown complete")
}
