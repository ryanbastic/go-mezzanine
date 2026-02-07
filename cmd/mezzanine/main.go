package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/ryanbastic/go-mezzanine/internal/api"
	"github.com/ryanbastic/go-mezzanine/internal/config"
	"github.com/ryanbastic/go-mezzanine/internal/index"
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
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: logLevel}))
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
		pool, err := pgxpool.New(ctx, b.DatabaseURL)
		if err != nil {
			logger.Error("failed to connect to backend", "backend", b.Name, "error", err)
			os.Exit(1)
		}
		if err := pool.Ping(ctx); err != nil {
			logger.Error("failed to ping backend", "backend", b.Name, "error", err)
			os.Exit(1)
		}
		pools[b.Name] = pool
		logger.Info("connected to backend", "backend", b.Name, "shards", []int{b.ShardStart, b.ShardEnd})
	}
	defer func() {
		for name, pool := range pools {
			pool.Close()
			logger.Info("closed pool", "backend", name)
		}
	}()

	logger.Info("running migrations")
	// Run migrations per backend
	for _, b := range shardCfg.Backends {
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
			s := storage.NewPostgresStore(pool, i)
			router.Register(shard.ID(i), s)
		}
	}

	// Initialize index registry
	indexRegistry := index.NewRegistry()

	if cfg.IndexConfigPath != "" {
		idxCfg, err := config.LoadIndexConfig(cfg.IndexConfigPath)
		if err != nil {
			logger.Error("failed to load index config", "error", err)
			os.Exit(1)
		}

		// Register all definitions across all backends
		for _, b := range shardCfg.Backends {
			pool := pools[b.Name]
			for _, idx := range idxCfg.Indexes {
				indexRegistry.RegisterRange(pool, index.Definition{
					Name:          idx.Name,
					SourceColumn:  idx.SourceColumn,
					ShardKeyField: idx.ShardKeyField,
					Fields:        idx.Fields,
				}, b.ShardStart, b.ShardEnd)
			}
		}

		// Create index tables per backend
		for _, b := range shardCfg.Backends {
			pool := pools[b.Name]
			if err := indexRegistry.CreateTablesRange(ctx, pool, b.ShardStart, b.ShardEnd); err != nil {
				logger.Error("failed to create index tables", "backend", b.Name, "error", err)
				os.Exit(1)
			}
			logger.Info("index tables created", "backend", b.Name, "shards", []int{b.ShardStart, b.ShardEnd})
		}

		logger.Info("indexes registered", "count", len(idxCfg.Indexes))
	}

	// Initialize trigger plugin system
	pluginRegistry := trigger.NewPluginRegistry()
	rpcClient := trigger.NewRPCClient(cfg.TriggerRetryMax, cfg.TriggerRetryBackoff, cfg.TriggerRPCTimeout)
	notifier := trigger.NewNotifier(pluginRegistry, rpcClient, logger)

	// Start HTTP server
	handler := api.NewServer(logger, router, indexRegistry, pluginRegistry, notifier, cfg.NumShards)
	srv := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: handler,
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
