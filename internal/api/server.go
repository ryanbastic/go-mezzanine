package api

import (
	"log/slog"
	"net/http"

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

	cellHandler := NewCellHandler(router, numShards)
	indexHandler := NewIndexHandler(indexRegistry, numShards)

	mux.Route("/v1", func(r chi.Router) {
		// Cell operations
		r.Post("/cells", cellHandler.WriteCell)
		r.Get("/cells/{row_key}/{column_name}/{ref_key}", cellHandler.GetCell)
		r.Get("/cells/{row_key}/{column_name}", cellHandler.GetCellLatest)
		r.Get("/cells/{row_key}", cellHandler.GetRow)

		// Secondary index queries
		r.Get("/index/{index_name}/{shard_key}", indexHandler.QueryIndex)

		// Health
		r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
			writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
		})
	})

	return mux
}
