package api

import (
	"log/slog"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"
	"github.com/ryanbastic/go-mezzanine/internal/index"
	"github.com/ryanbastic/go-mezzanine/internal/shard"
)

// NewServer creates an HTTP server with all routes configured.
func NewServer(logger *slog.Logger, router *shard.Router, indexRegistry *index.Registry, numShards int) http.Handler {
	mux := chi.NewRouter()

	mux.Use(RequestID)
	mux.Use(Logging(logger))
	mux.Use(Recovery(logger))

	config := huma.DefaultConfig("Mezzanine API", "1.0.0")
	config.Info.Description = "Sharded cell-based data store"
	api := humachi.New(mux, config)

	cellHandler := NewCellHandler(router, numShards, logger)
	indexHandler := NewIndexHandler(indexRegistry, numShards, logger)

	registerCellRoutes(api, cellHandler)
	registerIndexRoutes(api, indexHandler)
	registerHealthRoute(api)

	return mux
}
