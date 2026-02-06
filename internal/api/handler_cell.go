package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/ryanbastic/go-mezzanine/internal/cell"
	"github.com/ryanbastic/go-mezzanine/internal/shard"
	"github.com/ryanbastic/go-mezzanine/internal/storage"
)

// CellHandler handles HTTP requests for cell operations.
type CellHandler struct {
	router    *shard.Router
	numShards int
}

// NewCellHandler creates a new CellHandler.
func NewCellHandler(router *shard.Router, numShards int) *CellHandler {
	return &CellHandler{router: router, numShards: numShards}
}

// WriteCell handles POST /v1/cells
func (h *CellHandler) WriteCell(w http.ResponseWriter, r *http.Request) {
	var req cell.WriteCellRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.RowKey == uuid.Nil {
		writeError(w, http.StatusBadRequest, "row_key is required")
		return
	}
	if req.ColumnName == "" {
		writeError(w, http.StatusBadRequest, "column_name is required")
		return
	}
	if req.Body == nil {
		writeError(w, http.StatusBadRequest, "body is required")
		return
	}

	shardID := shard.ForRowKey(req.RowKey, h.numShards)
	store, err := h.router.StoreFor(shardID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "shard routing failed")
		return
	}

	c, err := store.WriteCell(r.Context(), req)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to write cell")
		return
	}

	writeJSON(w, http.StatusCreated, c)
}

// GetCell handles GET /v1/cells/{row_key}/{column_name}/{ref_key}
func (h *CellHandler) GetCell(w http.ResponseWriter, r *http.Request) {
	rowKey, err := uuid.Parse(chi.URLParam(r, "row_key"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid row_key")
		return
	}
	columnName := chi.URLParam(r, "column_name")
	refKey, err := strconv.ParseInt(chi.URLParam(r, "ref_key"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid ref_key")
		return
	}

	shardID := shard.ForRowKey(rowKey, h.numShards)
	store, err := h.router.StoreFor(shardID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "shard routing failed")
		return
	}

	ref := cell.CellRef{RowKey: rowKey, ColumnName: columnName, RefKey: refKey}
	c, err := store.GetCell(r.Context(), ref)
	if err != nil {
		if errors.Is(err, storage.ErrCellNotFound) {
			writeError(w, http.StatusNotFound, "cell not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to get cell")
		return
	}

	writeJSON(w, http.StatusOK, c)
}

// GetCellLatest handles GET /v1/cells/{row_key}/{column_name}
func (h *CellHandler) GetCellLatest(w http.ResponseWriter, r *http.Request) {
	rowKey, err := uuid.Parse(chi.URLParam(r, "row_key"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid row_key")
		return
	}
	columnName := chi.URLParam(r, "column_name")

	shardID := shard.ForRowKey(rowKey, h.numShards)
	store, err := h.router.StoreFor(shardID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "shard routing failed")
		return
	}

	c, err := store.GetCellLatest(r.Context(), rowKey, columnName)
	if err != nil {
		if errors.Is(err, storage.ErrCellNotFound) {
			writeError(w, http.StatusNotFound, "cell not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to get cell")
		return
	}

	writeJSON(w, http.StatusOK, c)
}

// GetRow handles GET /v1/cells/{row_key}
func (h *CellHandler) GetRow(w http.ResponseWriter, r *http.Request) {
	rowKey, err := uuid.Parse(chi.URLParam(r, "row_key"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid row_key")
		return
	}

	shardID := shard.ForRowKey(rowKey, h.numShards)
	store, err := h.router.StoreFor(shardID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "shard routing failed")
		return
	}

	cells, err := store.GetRow(r.Context(), rowKey)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get row")
		return
	}

	type rowResponse struct {
		RowKey uuid.UUID   `json:"row_key"`
		Cells  []cell.Cell `json:"cells"`
	}
	writeJSON(w, http.StatusOK, rowResponse{RowKey: rowKey, Cells: cells})
}
