package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/ryanbastic/go-mezzanine/internal/index"
	"github.com/ryanbastic/go-mezzanine/internal/shard"
)

// IndexHandler handles HTTP requests for secondary index queries.
type IndexHandler struct {
	registry  *index.Registry
	numShards int
}

// NewIndexHandler creates a new IndexHandler.
func NewIndexHandler(registry *index.Registry, numShards int) *IndexHandler {
	return &IndexHandler{registry: registry, numShards: numShards}
}

// QueryIndex handles GET /v1/index/{index_name}/{shard_key}
func (h *IndexHandler) QueryIndex(w http.ResponseWriter, r *http.Request) {
	indexName := chi.URLParam(r, "index_name")
	shardKey, err := uuid.Parse(chi.URLParam(r, "shard_key"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid shard_key")
		return
	}

	shardID := shard.ForRowKey(shardKey, h.numShards)
	store, ok := h.registry.StoreFor(indexName, shardID)
	if !ok {
		writeError(w, http.StatusNotFound, "index not found")
		return
	}

	entries, err := store.QueryByShardKey(r.Context(), shardKey)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to query index")
		return
	}

	writeJSON(w, http.StatusOK, entries)
}
