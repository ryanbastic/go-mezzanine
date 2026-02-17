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

// QueryIndexInput now supports pagination
type QueryIndexInput struct {
	IndexName string `path:"index_name" doc:"Secondary index name"`
	Value     string `path:"value" doc:"Lookup value (e.g. email address)" minLength:"1"`
	Limit     int    `query:"limit" doc:"Maximum number of entries to return (default 100, max 1000)" required:"false"`
	Cursor    string `query:"cursor" doc:"Opaque pagination cursor from previous response" required:"false"`
}

type IndexEntryResponse struct {
	AddedID   int64           `json:"added_id" doc:"Auto-incremented ID"`
	ShardKey  string          `json:"shard_key" doc:"Shard key value"`
	RowKey    uuid.UUID       `json:"row_key" doc:"Row key UUID"`
	Body      json.RawMessage `json:"body" doc:"Denormalized JSON payload"`
	CreatedAt time.Time       `json:"created_at" doc:"Creation timestamp"`
}

// QueryIndexOutput includes pagination metadata
type QueryIndexOutput struct {
	Entries    []IndexEntryResponse `json:"entries" doc:"Index entries in this page"`
	NextCursor string               `json:"next_cursor,omitempty" doc:"Cursor for next page (empty if no more results)"`
	HasMore    bool                 `json:"has_more" doc:"True if more results available"`
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
		Summary:     "Query secondary index with pagination",
		Tags:        []string{"index"},
	}, h.QueryIndex)
}

func (h *IndexHandler) QueryIndex(ctx context.Context, input *QueryIndexInput) (*QueryIndexOutput, error) {
	// Validate and cap limit
	limit := input.Limit
	if limit <= 0 {
		limit = 100 // default
	}
	if limit > 1000 {
		limit = 1000 // max
	}

	shardID := shard.ForKey(input.Value, h.numShards)
	store, ok := h.registry.StoreFor(input.IndexName, shardID)
	if !ok {
		return nil, huma.Error404NotFound("index not found")
	}

	page, err := store.QueryByShardKey(ctx, input.Value, input.Cursor, limit)
	if err != nil {
		h.logger.Error("failed to query index", "index_name", input.IndexName, "value", input.Value, "error", err)
		return nil, huma.Error500InternalServerError("failed to query index")
	}

	resp := make([]IndexEntryResponse, len(page.Entries))
	for i, e := range page.Entries {
		resp[i] = IndexEntryResponse{
			AddedID:   e.AddedID,
			ShardKey:  e.ShardKey,
			RowKey:    e.RowKey,
			Body:      e.Body,
			CreatedAt: e.CreatedAt,
		}
	}

	return &QueryIndexOutput{
		Entries:    resp,
		NextCursor: page.NextCursor,
		HasMore:    page.HasMore,
	}, nil
}
