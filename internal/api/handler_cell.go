package api

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"
	"github.com/ryanbastic/go-mezzanine/internal/cell"
	"github.com/ryanbastic/go-mezzanine/internal/shard"
	"github.com/ryanbastic/go-mezzanine/internal/storage"
)

// --- Huma Input/Output types ---

type WriteCellBody struct {
	RowKey     uuid.UUID       `json:"row_key" doc:"Row key UUID" required:"true"`
	ColumnName string          `json:"column_name" doc:"Column name" required:"true" minLength:"1"`
	RefKey     int64           `json:"ref_key" doc:"Reference key version"`
	Body       json.RawMessage `json:"body" doc:"Arbitrary JSON payload" required:"true"`
}

type WriteCellInput struct {
	Body WriteCellBody
}

type CellResponse struct {
	AddedID    int64           `json:"added_id" doc:"Auto-incremented ID"`
	RowKey     uuid.UUID       `json:"row_key" doc:"Row key UUID"`
	ColumnName string          `json:"column_name" doc:"Column name"`
	RefKey     int64           `json:"ref_key" doc:"Reference key version"`
	Body       json.RawMessage `json:"body" doc:"Stored JSON payload"`
	CreatedAt  time.Time       `json:"created_at" doc:"Creation timestamp"`
}

type WriteCellOutput struct {
	Body CellResponse
}

type GetCellInput struct {
	RowKey     string `path:"row_key" doc:"Row key UUID" format:"uuid"`
	ColumnName string `path:"column_name" doc:"Column name"`
	RefKey     int64  `path:"ref_key" doc:"Reference key version"`
}

type GetCellOutput struct {
	Body CellResponse
}

type GetCellLatestInput struct {
	RowKey     string `path:"row_key" doc:"Row key UUID" format:"uuid"`
	ColumnName string `path:"column_name" doc:"Column name"`
}

type GetCellLatestOutput struct {
	Body CellResponse
}

type GetRowInput struct {
	RowKey string `path:"row_key" doc:"Row key UUID" format:"uuid"`
}

type RowResponse struct {
	RowKey uuid.UUID      `json:"row_key" doc:"Row key UUID"`
	Cells  []CellResponse `json:"cells" doc:"Latest cell per column"`
}

type GetRowOutput struct {
	Body RowResponse
}

// --- Handler ---

type CellHandler struct {
	router    *shard.Router
	numShards int
	logger    *slog.Logger
}

func NewCellHandler(router *shard.Router, numShards int, logger *slog.Logger) *CellHandler {
	return &CellHandler{router: router, numShards: numShards, logger: logger}
}

func registerCellRoutes(api huma.API, h *CellHandler) {
	huma.Register(api, huma.Operation{
		OperationID:   "write-cell",
		Method:        http.MethodPost,
		Path:          "/v1/cells",
		Summary:       "Write a cell",
		Tags:          []string{"cells"},
		DefaultStatus: http.StatusCreated,
	}, h.WriteCell)

	huma.Register(api, huma.Operation{
		OperationID: "get-cell",
		Method:      http.MethodGet,
		Path:        "/v1/cells/{row_key}/{column_name}/{ref_key}",
		Summary:     "Get exact cell version",
		Tags:        []string{"cells"},
	}, h.GetCell)

	huma.Register(api, huma.Operation{
		OperationID: "get-cell-latest",
		Method:      http.MethodGet,
		Path:        "/v1/cells/{row_key}/{column_name}",
		Summary:     "Get latest cell version",
		Tags:        []string{"cells"},
	}, h.GetCellLatest)

	huma.Register(api, huma.Operation{
		OperationID: "get-row",
		Method:      http.MethodGet,
		Path:        "/v1/cells/{row_key}",
		Summary:     "Get all latest cells for a row",
		Tags:        []string{"cells"},
	}, h.GetRow)
}

func (h *CellHandler) WriteCell(ctx context.Context, input *WriteCellInput) (*WriteCellOutput, error) {
	req := cell.WriteCellRequest{
		RowKey:     input.Body.RowKey,
		ColumnName: input.Body.ColumnName,
		RefKey:     input.Body.RefKey,
		Body:       input.Body.Body,
	}

	shardID := shard.ForRowKey(req.RowKey, h.numShards)
	store, err := h.router.StoreFor(shardID)
	if err != nil {
		h.logger.Error("shard routing failed", "shard_id", shardID, "error", err)
		return nil, huma.Error500InternalServerError("shard routing failed")
	}

	c, err := store.WriteCell(ctx, req)
	if err != nil {
		h.logger.Error("failed to write cell", "row_key", req.RowKey, "column_name", req.ColumnName, "error", err)
		return nil, huma.Error500InternalServerError("failed to write cell")
	}

	return &WriteCellOutput{Body: cellToResponse(c)}, nil
}

func (h *CellHandler) GetCell(ctx context.Context, input *GetCellInput) (*GetCellOutput, error) {
	rowKey, err := uuid.Parse(input.RowKey)
	if err != nil {
		return nil, huma.Error400BadRequest("invalid row_key")
	}

	shardID := shard.ForRowKey(rowKey, h.numShards)
	store, err := h.router.StoreFor(shardID)
	if err != nil {
		h.logger.Error("shard routing failed", "shard_id", shardID, "error", err)
		return nil, huma.Error500InternalServerError("shard routing failed")
	}

	ref := cell.CellRef{RowKey: rowKey, ColumnName: input.ColumnName, RefKey: input.RefKey}
	c, err := store.GetCell(ctx, ref)
	if err != nil {
		if errors.Is(err, storage.ErrCellNotFound) {
			return nil, huma.Error404NotFound("cell not found")
		}
		h.logger.Error("failed to get cell", "row_key", rowKey, "column_name", input.ColumnName, "ref_key", input.RefKey, "error", err)
		return nil, huma.Error500InternalServerError("failed to get cell")
	}

	return &GetCellOutput{Body: cellToResponse(c)}, nil
}

func (h *CellHandler) GetCellLatest(ctx context.Context, input *GetCellLatestInput) (*GetCellLatestOutput, error) {
	rowKey, err := uuid.Parse(input.RowKey)
	if err != nil {
		return nil, huma.Error400BadRequest("invalid row_key")
	}

	shardID := shard.ForRowKey(rowKey, h.numShards)
	store, err := h.router.StoreFor(shardID)
	if err != nil {
		h.logger.Error("shard routing failed", "shard_id", shardID, "error", err)
		return nil, huma.Error500InternalServerError("shard routing failed")
	}

	c, err := store.GetCellLatest(ctx, rowKey, input.ColumnName)
	if err != nil {
		if errors.Is(err, storage.ErrCellNotFound) {
			return nil, huma.Error404NotFound("cell not found")
		}
		h.logger.Error("failed to get cell", "row_key", rowKey, "column_name", input.ColumnName, "error", err)
		return nil, huma.Error500InternalServerError("failed to get cell")
	}

	return &GetCellLatestOutput{Body: cellToResponse(c)}, nil
}

func (h *CellHandler) GetRow(ctx context.Context, input *GetRowInput) (*GetRowOutput, error) {
	rowKey, err := uuid.Parse(input.RowKey)
	if err != nil {
		return nil, huma.Error400BadRequest("invalid row_key")
	}

	shardID := shard.ForRowKey(rowKey, h.numShards)
	store, err := h.router.StoreFor(shardID)
	if err != nil {
		h.logger.Error("shard routing failed", "shard_id", shardID, "error", err)
		return nil, huma.Error500InternalServerError("shard routing failed")
	}

	cells, err := store.GetRow(ctx, rowKey)
	if err != nil {
		h.logger.Error("failed to get row", "row_key", rowKey, "error", err)
		return nil, huma.Error500InternalServerError("failed to get row")
	}

	resp := make([]CellResponse, len(cells))
	for i, c := range cells {
		resp[i] = CellResponse{
			AddedID:    c.AddedID,
			RowKey:     c.RowKey,
			ColumnName: c.ColumnName,
			RefKey:     c.RefKey,
			Body:       c.Body,
			CreatedAt:  c.CreatedAt,
		}
	}

	return &GetRowOutput{Body: RowResponse{RowKey: rowKey, Cells: resp}}, nil
}

func cellToResponse(c *cell.Cell) CellResponse {
	return CellResponse{
		AddedID:    c.AddedID,
		RowKey:     c.RowKey,
		ColumnName: c.ColumnName,
		RefKey:     c.RefKey,
		Body:       c.Body,
		CreatedAt:  c.CreatedAt,
	}
}
