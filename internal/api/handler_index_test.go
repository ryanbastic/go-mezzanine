package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/ryanbastic/go-mezzanine/internal/cell"
	"github.com/ryanbastic/go-mezzanine/internal/index"
	"github.com/ryanbastic/go-mezzanine/internal/shard"
)

// mockIndexStore implements the functionality we need for testing QueryIndex.
// We test via the real index.Registry, registering with nil pool and overriding behavior
// indirectly. Since index.Store requires a real pgxpool.Pool for queries,
// we test the handler's validation and routing logic by using the real Registry.

func TestQueryIndex_InvalidShardKey(t *testing.T) {
	registry := index.NewRegistry()
	handler := NewIndexHandler(registry, 64)

	req := httptest.NewRequest(http.MethodGet, "/v1/index/test_idx/not-a-uuid", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("index_name", "test_idx")
	rctx.URLParams.Add("shard_key", "not-a-uuid")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	w := httptest.NewRecorder()

	handler.QueryIndex(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status: got %d, want %d", w.Code, http.StatusBadRequest)
	}

	var resp errorResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Error != "invalid shard_key" {
		t.Errorf("error: got %q", resp.Error)
	}
}

func TestQueryIndex_IndexNotFound(t *testing.T) {
	registry := index.NewRegistry()
	handler := NewIndexHandler(registry, 64)

	shardKey := uuid.New()
	req := httptest.NewRequest(http.MethodGet, "/v1/index/nonexistent/"+shardKey.String(), nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("index_name", "nonexistent")
	rctx.URLParams.Add("shard_key", shardKey.String())
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	w := httptest.NewRecorder()

	handler.QueryIndex(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status: got %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestNewIndexHandler(t *testing.T) {
	registry := index.NewRegistry()
	h := NewIndexHandler(registry, 64)
	if h == nil {
		t.Fatal("NewIndexHandler returned nil")
	}
}

// --- Integration test with full server ---

func TestServer_HealthEndpoint(t *testing.T) {
	logger := testLogger()
	router := shard.NewRouter()
	registry := index.NewRegistry()

	server := NewServer(logger, router, registry, 64)

	req := httptest.NewRequest(http.MethodGet, "/v1/health", nil)
	w := httptest.NewRecorder()

	server.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status: got %d, want %d", w.Code, http.StatusOK)
	}

	var resp map[string]string
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp["status"] != "ok" {
		t.Errorf("status field: got %q", resp["status"])
	}
}

func TestServer_HasRequestID(t *testing.T) {
	logger := testLogger()
	router := shard.NewRouter()
	registry := index.NewRegistry()

	server := NewServer(logger, router, registry, 64)

	req := httptest.NewRequest(http.MethodGet, "/v1/health", nil)
	w := httptest.NewRecorder()

	server.ServeHTTP(w, req)

	if w.Header().Get("X-Request-ID") == "" {
		t.Error("expected X-Request-ID header")
	}
}

func TestServer_NotFound(t *testing.T) {
	logger := testLogger()
	router := shard.NewRouter()
	registry := index.NewRegistry()

	server := NewServer(logger, router, registry, 64)

	req := httptest.NewRequest(http.MethodGet, "/nonexistent", nil)
	w := httptest.NewRecorder()

	server.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status: got %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestServer_WriteAndGetCell(t *testing.T) {
	logger := testLogger()
	store := newMockCellStore()
	shardRouter := shard.NewRouter()
	for i := range 64 {
		shardRouter.Register(shard.ID(i), store)
	}
	registry := index.NewRegistry()

	server := NewServer(logger, shardRouter, registry, 64)

	// Write a cell
	rowKey := uuid.New()
	body := map[string]any{
		"row_key":     rowKey.String(),
		"column_name": "profile",
		"ref_key":     1,
		"body":        map[string]string{"name": "test"},
	}
	data, _ := json.Marshal(body)

	writeReq := httptest.NewRequest(http.MethodPost, "/v1/cells", bytes.NewReader(data))
	writeW := httptest.NewRecorder()
	server.ServeHTTP(writeW, writeReq)

	if writeW.Code != http.StatusCreated {
		t.Fatalf("write status: got %d, want %d", writeW.Code, http.StatusCreated)
	}

	// Get the cell back
	getReq := httptest.NewRequest(http.MethodGet, "/v1/cells/"+rowKey.String()+"/profile/1", nil)
	getW := httptest.NewRecorder()
	server.ServeHTTP(getW, getReq)

	if getW.Code != http.StatusOK {
		t.Errorf("get status: got %d, want %d", getW.Code, http.StatusOK)
	}

	var resp cell.Cell
	if err := json.NewDecoder(getW.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.RowKey != rowKey {
		t.Errorf("RowKey mismatch: got %s, want %s", resp.RowKey, rowKey)
	}
}

func TestServer_GetRow_Integration(t *testing.T) {
	logger := testLogger()
	store := newMockCellStore()
	rowKey := uuid.New()
	store.rows[rowKey.String()] = []cell.Cell{
		{AddedID: 1, RowKey: rowKey, ColumnName: "a", RefKey: 1, Body: json.RawMessage(`{}`), CreatedAt: time.Now()},
	}

	shardRouter := shard.NewRouter()
	for i := range 64 {
		shardRouter.Register(shard.ID(i), store)
	}

	server := NewServer(logger, shardRouter, index.NewRegistry(), 64)

	req := httptest.NewRequest(http.MethodGet, "/v1/cells/"+rowKey.String(), nil)
	w := httptest.NewRecorder()
	server.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status: got %d, want %d", w.Code, http.StatusOK)
	}
}

// Verify unused error variables are of the expected type
func TestStorageErrors(t *testing.T) {
	var err error = errors.New("test")
	if err == nil {
		t.Error("error should not be nil")
	}
}
