package api

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"
	"github.com/ryanbastic/go-mezzanine/internal/index"
	"github.com/ryanbastic/go-mezzanine/internal/shard"
	"github.com/ryanbastic/go-mezzanine/internal/trigger"
)

// NewServer creates an HTTP server with all routes configured.
// backends maps backend names to Pinger instances (e.g. *pgxpool.Pool) for
// readiness checks. Pass nil when backends are not available (e.g. in tests).
func NewServer(logger *slog.Logger, router *shard.Router, indexRegistry *index.Registry, pluginRegistry *trigger.PluginRegistry, notifier *trigger.Notifier, numShards int, backends map[string]Pinger) http.Handler {
	mux := chi.NewRouter()

	mux.Use(RequestID)
	mux.Use(Logging(logger))
	mux.Use(Recovery(logger))

	// Health probes registered directly on Chi (need conditional status codes).
	healthHandler := NewHealthHandler(backends, logger)
	mux.Get("/v1/livez", healthHandler.Livez)
	mux.Get("/v1/readyz", healthHandler.Readyz)
	mux.Get("/v1/health", healthHandler.Readyz)

	config := huma.DefaultConfig("Mezzanine API", "1.0.0")
	config.Info.Description = "Sharded cell-based data store"
	api := humachi.New(mux, config)

	cellHandler := NewCellHandler(router, numShards, indexRegistry, notifier, logger)
	indexHandler := NewIndexHandler(indexRegistry, numShards, logger)
	pluginHandler := NewPluginHandler(pluginRegistry, logger)

	registerCellRoutes(api, cellHandler)
	registerIndexRoutes(api, indexHandler)
	registerPluginRoutes(api, pluginHandler)
	registerShardRoutes(api, numShards)

	return mux
}

// --- Shard Info ---

type ShardCountInput struct{}

type ShardCountResponse struct {
	NumShards int `json:"num_shards" doc:"Number of configured shards" example:"64"`
}

type ShardCountOutput struct {
	Body ShardCountResponse
}

func registerShardRoutes(api huma.API, numShards int) {
	huma.Register(api, huma.Operation{
		OperationID: "get-shard-count",
		Method:      http.MethodGet,
		Path:        "/v1/shards/count",
		Summary:     "Get shard count",
		Tags:        []string{"shards"},
	}, func(ctx context.Context, input *ShardCountInput) (*ShardCountOutput, error) {
		return &ShardCountOutput{Body: ShardCountResponse{NumShards: numShards}}, nil
	})
}
