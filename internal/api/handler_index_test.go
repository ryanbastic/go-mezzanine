package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/ryanbastic/go-mezzanine/internal/cell"
	"github.com/ryanbastic/go-mezzanine/internal/index"
	"github.com/ryanbastic/go-mezzanine/internal/shard"
	"github.com/ryanbastic/go-mezzanine/internal/trigger"
)

func TestQueryIndex_InvalidShardKey(t *testing.T) {
	server := NewServer(testLogger(), shard.NewRouter(), index.NewRegistry(), trigger.NewPluginRegistry(), nil, 64)

	req := httptest.NewRequest(http.MethodGet, "/v1/index/test_idx/not-a-uuid", nil)
	w := httptest.NewRecorder()

	server.ServeHTTP(w, req)

	if w.Code < 400 || w.Code >= 500 {
		t.Errorf("status: got %d, want 4xx\nbody: %s", w.Code, w.Body.String())
	}
}

func TestQueryIndex_IndexNotFound(t *testing.T) {
	server := NewServer(testLogger(), shard.NewRouter(), index.NewRegistry(), trigger.NewPluginRegistry(), nil, 64)

	shardKey := uuid.New()
	req := httptest.NewRequest(http.MethodGet, "/v1/index/nonexistent/"+shardKey.String(), nil)
	w := httptest.NewRecorder()

	server.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status: got %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestNewIndexHandler(t *testing.T) {
	registry := index.NewRegistry()
	h := NewIndexHandler(registry, 64, testLogger())
	if h == nil {
		t.Fatal("NewIndexHandler returned nil")
	}
}

// --- Integration tests ---

func TestServer_HealthEndpoint(t *testing.T) {
	server := NewServer(testLogger(), shard.NewRouter(), index.NewRegistry(), trigger.NewPluginRegistry(), nil, 64)

	req := httptest.NewRequest(http.MethodGet, "/v1/health", nil)
	w := httptest.NewRecorder()

	server.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status: got %d, want %d\nbody: %s", w.Code, http.StatusOK, w.Body.String())
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
	server := NewServer(testLogger(), shard.NewRouter(), index.NewRegistry(), trigger.NewPluginRegistry(), nil, 64)

	req := httptest.NewRequest(http.MethodGet, "/v1/health", nil)
	w := httptest.NewRecorder()

	server.ServeHTTP(w, req)

	if w.Header().Get("X-Request-ID") == "" {
		t.Error("expected X-Request-ID header")
	}
}

func TestServer_NotFound(t *testing.T) {
	server := NewServer(testLogger(), shard.NewRouter(), index.NewRegistry(), trigger.NewPluginRegistry(), nil, 64)

	req := httptest.NewRequest(http.MethodGet, "/nonexistent", nil)
	w := httptest.NewRecorder()

	server.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status: got %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestServer_WriteAndGetCell(t *testing.T) {
	store := newMockCellStore()
	shardRouter := shard.NewRouter()
	for i := range 64 {
		shardRouter.Register(shard.ID(i), store)
	}

	server := NewServer(testLogger(), shardRouter, index.NewRegistry(), trigger.NewPluginRegistry(), nil, 64)

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
	writeReq.Header.Set("Content-Type", "application/json")
	writeW := httptest.NewRecorder()
	server.ServeHTTP(writeW, writeReq)

	if writeW.Code != http.StatusCreated {
		t.Fatalf("write status: got %d, want %d\nbody: %s", writeW.Code, http.StatusCreated, writeW.Body.String())
	}

	// Get the cell back
	getReq := httptest.NewRequest(http.MethodGet, "/v1/cells/"+rowKey.String()+"/profile/1", nil)
	getW := httptest.NewRecorder()
	server.ServeHTTP(getW, getReq)

	if getW.Code != http.StatusOK {
		t.Errorf("get status: got %d, want %d\nbody: %s", getW.Code, http.StatusOK, getW.Body.String())
	}

	var resp CellResponse
	if err := json.NewDecoder(getW.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.RowKey != rowKey {
		t.Errorf("RowKey mismatch: got %s, want %s", resp.RowKey, rowKey)
	}
}

func TestServer_GetRow_Integration(t *testing.T) {
	store := newMockCellStore()
	rowKey := uuid.New()
	store.rows[rowKey.String()] = []cell.Cell{
		{AddedID: 1, RowKey: rowKey, ColumnName: "a", RefKey: 1, Body: json.RawMessage(`{}`), CreatedAt: time.Now()},
	}

	shardRouter := shard.NewRouter()
	for i := range 64 {
		shardRouter.Register(shard.ID(i), store)
	}

	server := NewServer(testLogger(), shardRouter, index.NewRegistry(), trigger.NewPluginRegistry(), nil, 64)

	req := httptest.NewRequest(http.MethodGet, "/v1/cells/"+rowKey.String(), nil)
	w := httptest.NewRecorder()
	server.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status: got %d, want %d", w.Code, http.StatusOK)
	}
}

func TestServer_OpenAPISpec(t *testing.T) {
	server := NewServer(testLogger(), shard.NewRouter(), index.NewRegistry(), trigger.NewPluginRegistry(), nil, 64)

	req := httptest.NewRequest(http.MethodGet, "/openapi.json", nil)
	w := httptest.NewRecorder()

	server.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status: got %d, want %d", w.Code, http.StatusOK)
	}

	var spec map[string]any
	if err := json.NewDecoder(w.Body).Decode(&spec); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if spec["openapi"] == nil {
		t.Error("expected openapi field in spec")
	}
}
