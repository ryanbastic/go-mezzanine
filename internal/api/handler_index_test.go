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

	"github.com/google/uuid"
	"github.com/ryanbastic/go-mezzanine/internal/cell"
	"github.com/ryanbastic/go-mezzanine/internal/index"
	"github.com/ryanbastic/go-mezzanine/internal/shard"
	"github.com/ryanbastic/go-mezzanine/internal/trigger"
)

// --- Mock IndexStore ---

type mockIndexStore struct {
	entries  []index.Entry
	queryErr error
	writeErr error
}

func (m *mockIndexStore) QueryByShardKey(_ context.Context, _ string) ([]index.Entry, error) {
	if m.queryErr != nil {
		return nil, m.queryErr
	}
	return m.entries, nil
}

func (m *mockIndexStore) WriteEntry(_ context.Context, entry index.Entry) error {
	if m.writeErr != nil {
		return m.writeErr
	}
	m.entries = append(m.entries, entry)
	return nil
}

func setupIndexTestServer(mockStore index.IndexStore, indexName string, numShards int) http.Handler {
	registry := index.NewRegistry()
	for i := range numShards {
		registry.RegisterStore(indexName, shard.ID(i), mockStore)
	}
	return NewServer(testLogger(), shard.NewRouter(), registry, trigger.NewPluginRegistry(), nil, numShards, nil)
}

func TestQueryIndex_IndexNotFound(t *testing.T) {
	server := NewServer(testLogger(), shard.NewRouter(), index.NewRegistry(), trigger.NewPluginRegistry(), nil, 64, nil)

	req := httptest.NewRequest(http.MethodGet, "/v1/index/nonexistent/alice@example.com", nil)
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

// --- user_by_email integration tests ---

func TestQueryIndex_UserByEmail_FoundRoute(t *testing.T) {
	// Register user_by_email index definition so routing resolves (not 404).
	registry := index.NewRegistry()
	registry.Register(nil, index.Definition{
		Name:          "user_by_email",
		SourceColumn:  "profile",
		ShardKeyField: "email",
		Fields:        []string{"email", "display_name"},
		UniqueFields:  []string{"email"},
	}, 64)

	server := NewServer(testLogger(), shard.NewRouter(), registry, trigger.NewPluginRegistry(), nil, 64, nil)

	req := httptest.NewRequest(http.MethodGet, "/v1/index/user_by_email/alice@example.com", nil)
	w := httptest.NewRecorder()

	server.ServeHTTP(w, req)

	// Store has nil pool so QueryByShardKey will fail with 500,
	// but the route resolved — NOT 404. This proves the index is registered.
	if w.Code == http.StatusNotFound {
		t.Errorf("expected route to resolve (not 404), got %d", w.Code)
	}
}

func TestWriteCell_UserByEmail_ProfileStored(t *testing.T) {
	store := newMockCellStore()
	shardRouter := shard.NewRouter()
	for i := range 64 {
		shardRouter.Register(shard.ID(i), store)
	}

	// No index registry — just verify profile cell with email is stored correctly.
	server := NewServer(testLogger(), shardRouter, index.NewRegistry(), trigger.NewPluginRegistry(), nil, 64, nil)

	rowKey := uuid.New()
	body := map[string]any{
		"row_key":     rowKey.String(),
		"column_name": "profile",
		"ref_key":     1,
		"body": map[string]any{
			"email":        "alice@example.com",
			"display_name": "Alice Smith",
		},
	}
	data, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/v1/cells", bytes.NewReader(data))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("status: got %d, want %d\nbody: %s", w.Code, http.StatusCreated, w.Body.String())
	}

	var resp CellResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.RowKey != rowKey {
		t.Errorf("RowKey: got %s, want %s", resp.RowKey, rowKey)
	}
	if resp.ColumnName != "profile" {
		t.Errorf("ColumnName: got %q, want %q", resp.ColumnName, "profile")
	}

	// Verify the stored body contains the email field.
	var storedBody map[string]any
	if err := json.Unmarshal(resp.Body, &storedBody); err != nil {
		t.Fatalf("unmarshal body: %v", err)
	}
	if storedBody["email"] != "alice@example.com" {
		t.Errorf("email: got %v", storedBody["email"])
	}
}

// --- Integration tests ---

func TestServer_HasRequestID(t *testing.T) {
	server := NewServer(testLogger(), shard.NewRouter(), index.NewRegistry(), trigger.NewPluginRegistry(), nil, 64, nil)

	req := httptest.NewRequest(http.MethodGet, "/v1/health", nil)
	w := httptest.NewRecorder()

	server.ServeHTTP(w, req)

	if w.Header().Get("X-Request-ID") == "" {
		t.Error("expected X-Request-ID header")
	}
}

func TestServer_NotFound(t *testing.T) {
	server := NewServer(testLogger(), shard.NewRouter(), index.NewRegistry(), trigger.NewPluginRegistry(), nil, 64, nil)

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

	server := NewServer(testLogger(), shardRouter, index.NewRegistry(), trigger.NewPluginRegistry(), nil, 64, nil)

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

	server := NewServer(testLogger(), shardRouter, index.NewRegistry(), trigger.NewPluginRegistry(), nil, 64, nil)

	req := httptest.NewRequest(http.MethodGet, "/v1/cells/"+rowKey.String(), nil)
	w := httptest.NewRecorder()
	server.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status: got %d, want %d", w.Code, http.StatusOK)
	}
}

// --- QueryIndex mock-backed tests ---

func TestQueryIndex_Success(t *testing.T) {
	rowKey := uuid.New()
	now := time.Now().Truncate(time.Microsecond)

	mock := &mockIndexStore{
		entries: []index.Entry{
			{
				AddedID:   1,
				ShardKey:  "alice@example.com",
				RowKey:    rowKey,
				Body:      json.RawMessage(`{"email":"alice@example.com","display_name":"Alice Smith"}`),
				CreatedAt: now,
			},
		},
	}

	server := setupIndexTestServer(mock, "user_by_email", 64)

	req := httptest.NewRequest(http.MethodGet, "/v1/index/user_by_email/alice@example.com", nil)
	w := httptest.NewRecorder()
	server.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status: got %d, want %d\nbody: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp []IndexEntryResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(resp) != 1 {
		t.Fatalf("entries: got %d, want 1", len(resp))
	}
	if resp[0].ShardKey != "alice@example.com" {
		t.Errorf("ShardKey: got %q, want %q", resp[0].ShardKey, "alice@example.com")
	}
	if resp[0].RowKey != rowKey {
		t.Errorf("RowKey: got %s, want %s", resp[0].RowKey, rowKey)
	}
}

func TestQueryIndex_EmptyResults(t *testing.T) {
	mock := &mockIndexStore{entries: []index.Entry{}}

	server := setupIndexTestServer(mock, "user_by_email", 64)

	req := httptest.NewRequest(http.MethodGet, "/v1/index/user_by_email/nobody@example.com", nil)
	w := httptest.NewRecorder()
	server.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status: got %d, want %d\nbody: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp []IndexEntryResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(resp) != 0 {
		t.Errorf("entries: got %d, want 0", len(resp))
	}
}

func TestQueryIndex_StoreError(t *testing.T) {
	mock := &mockIndexStore{queryErr: errors.New("db connection failed")}

	server := setupIndexTestServer(mock, "user_by_email", 64)

	req := httptest.NewRequest(http.MethodGet, "/v1/index/user_by_email/alice@example.com", nil)
	w := httptest.NewRecorder()
	server.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status: got %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

func TestServer_OpenAPISpec(t *testing.T) {
	server := NewServer(testLogger(), shard.NewRouter(), index.NewRegistry(), trigger.NewPluginRegistry(), nil, 64, nil)

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
