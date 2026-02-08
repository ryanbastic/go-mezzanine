package api

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"
	"github.com/ryanbastic/go-mezzanine/internal/index"
	"github.com/ryanbastic/go-mezzanine/internal/shard"
)

// --- Huma Input/Output types ---

type QueryIndexInput struct {
	IndexName string `path:"index_name" doc:"Secondary index name"`
	Value     string `path:"value" doc:"Lookup value (e.g. email address)" minLength:"1"`
}

type IndexEntryResponse struct {
	AddedID   int64           `json:"added_id" doc:"Auto-incremented ID"`
	ShardKey  string          `json:"shard_key" doc:"Shard key value"`
	RowKey    uuid.UUID       `json:"row_key" doc:"Row key UUID"`
	Body      json.RawMessage `json:"body" doc:"Denormalized JSON payload"`
	CreatedAt time.Time       `json:"created_at" doc:"Creation timestamp"`
}

type QueryIndexOutput struct {
	Body []IndexEntryResponse
}

// --- Handler ---

type IndexHandler struct {
	registry  *index.Registry
	numShards int
	logger    *slog.Logger
}

func NewIndexHandler(registry *index.Registry, numShards int, logger *slog.Logger) *IndexHandler {
	return &IndexHandler{registry: registry, numShards: numShards, logger: logger}
}

func registerIndexRoutes(api huma.API, h *IndexHandler) {
	huma.Register(api, huma.Operation{
		OperationID: "query-index",
		Method:      http.MethodGet,
		Path:        "/v1/index/{index_name}/{value}",
		Summary:     "Query secondary index",
		Tags:        []string{"index"},
	}, h.QueryIndex)
}

func (h *IndexHandler) QueryIndex(ctx context.Context, input *QueryIndexInput) (*QueryIndexOutput, error) {
	shardID := shard.ForKey(input.Value, h.numShards)
	store, ok := h.registry.StoreFor(input.IndexName, shardID)
	if !ok {
		return nil, huma.Error404NotFound("index not found")
	}

	entries, err := store.QueryByShardKey(ctx, input.Value)
	if err != nil {
		h.logger.Error("failed to query index", "index_name", input.IndexName, "value", input.Value, "error", err)
		return nil, huma.Error500InternalServerError("failed to query index")
	}

	resp := make([]IndexEntryResponse, len(entries))
	for i, e := range entries {
		resp[i] = IndexEntryResponse{
			AddedID:   e.AddedID,
			ShardKey:  e.ShardKey,
			RowKey:    e.RowKey,
			Body:      e.Body,
			CreatedAt: e.CreatedAt,
		}
	}

	return &QueryIndexOutput{Body: resp}, nil
}

// --- Health ---

type HealthInput struct{}

type HealthResponse struct {
	Status string `json:"status" doc:"Service health status" example:"ok"`
}

type HealthOutput struct {
	Body HealthResponse
}

func registerHealthRoute(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "health-check",
		Method:      http.MethodGet,
		Path:        "/v1/health",
		Summary:     "Health check",
		Tags:        []string{"health"},
	}, func(ctx context.Context, input *HealthInput) (*HealthOutput, error) {
		return &HealthOutput{Body: HealthResponse{Status: "ok"}}, nil
	})
}
