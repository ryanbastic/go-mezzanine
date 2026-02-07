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
	"github.com/ryanbastic/go-mezzanine/internal/storage"
	"github.com/ryanbastic/go-mezzanine/internal/trigger"
)

// --- Mock CellStore ---

type mockCellStore struct {
	cells     map[string]*cell.Cell
	rows      map[string][]cell.Cell
	writeErr  error
	getErr    error
	latestErr error
	rowErr    error
	nextID    int64
}

func newMockCellStore() *mockCellStore {
	return &mockCellStore{
		cells: make(map[string]*cell.Cell),
		rows:  make(map[string][]cell.Cell),
	}
}

func cellKey(rowKey uuid.UUID, colName string, refKey int64) string {
	return rowKey.String() + ":" + colName + ":" + string(rune(refKey))
}

func (m *mockCellStore) WriteCell(ctx context.Context, req cell.WriteCellRequest) (*cell.Cell, error) {
	if m.writeErr != nil {
		return nil, m.writeErr
	}
	m.nextID++
	c := &cell.Cell{
		AddedID:    m.nextID,
		RowKey:     req.RowKey,
		ColumnName: req.ColumnName,
		RefKey:     req.RefKey,
		Body:       req.Body,
		CreatedAt:  time.Now(),
	}
	m.cells[cellKey(req.RowKey, req.ColumnName, req.RefKey)] = c
	return c, nil
}

func (m *mockCellStore) GetCell(ctx context.Context, ref cell.CellRef) (*cell.Cell, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	c, ok := m.cells[cellKey(ref.RowKey, ref.ColumnName, ref.RefKey)]
	if !ok {
		return nil, storage.ErrCellNotFound
	}
	return c, nil
}

func (m *mockCellStore) GetCellLatest(ctx context.Context, rowKey uuid.UUID, columnName string) (*cell.Cell, error) {
	if m.latestErr != nil {
		return nil, m.latestErr
	}
	var best *cell.Cell
	for _, c := range m.cells {
		if c.RowKey == rowKey && c.ColumnName == columnName {
			if best == nil || c.RefKey > best.RefKey {
				cc := *c
				best = &cc
			}
		}
	}
	if best == nil {
		return nil, storage.ErrCellNotFound
	}
	return best, nil
}

func (m *mockCellStore) GetRow(ctx context.Context, rowKey uuid.UUID) ([]cell.Cell, error) {
	if m.rowErr != nil {
		return nil, m.rowErr
	}
	return m.rows[rowKey.String()], nil
}

func (m *mockCellStore) PartitionRead(ctx context.Context, partitionNumber int, readType int, addedID int64, createdAfter time.Time, limit int) ([]cell.Cell, error) {
	return nil, nil
}

func (m *mockCellStore) ScanCells(ctx context.Context, columnName string, afterAddedID int64, limit int) ([]cell.Cell, error) {
	return nil, nil
}

func setupTestServer(store storage.CellStore, numShards int) http.Handler {
	r := shard.NewRouter()
	for i := 0; i < numShards; i++ {
		r.Register(shard.ID(i), store)
	}
	return NewServer(testLogger(), r, index.NewRegistry(), trigger.NewPluginRegistry(), nil, numShards)
}

// --- WriteCell Tests ---

func TestWriteCell_Success(t *testing.T) {
	store := newMockCellStore()
	server := setupTestServer(store, 64)

	rowKey := uuid.New()
	body := map[string]any{
		"row_key":     rowKey.String(),
		"column_name": "profile",
		"ref_key":     1,
		"body":        map[string]string{"name": "test"},
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
		t.Errorf("ColumnName: got %q", resp.ColumnName)
	}
}

func TestWriteCell_InvalidBody(t *testing.T) {
	store := newMockCellStore()
	server := setupTestServer(store, 64)

	req := httptest.NewRequest(http.MethodPost, "/v1/cells", bytes.NewReader([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status: got %d, want %d\nbody: %s", w.Code, http.StatusBadRequest, w.Body.String())
	}
}

func TestWriteCell_MissingColumnName(t *testing.T) {
	store := newMockCellStore()
	server := setupTestServer(store, 64)

	body := map[string]any{
		"row_key": uuid.New().String(),
		"ref_key": 1,
		"body":    map[string]string{"name": "test"},
	}
	data, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/v1/cells", bytes.NewReader(data))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.ServeHTTP(w, req)

	if w.Code < 400 || w.Code >= 500 {
		t.Errorf("status: got %d, want 4xx\nbody: %s", w.Code, w.Body.String())
	}
}

func TestWriteCell_StoreError(t *testing.T) {
	store := newMockCellStore()
	store.writeErr = errors.New("db error")
	server := setupTestServer(store, 64)

	body := map[string]any{
		"row_key":     uuid.New().String(),
		"column_name": "profile",
		"ref_key":     1,
		"body":        map[string]string{"name": "test"},
	}
	data, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/v1/cells", bytes.NewReader(data))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status: got %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

// --- GetCell Tests ---

func TestGetCell_Success(t *testing.T) {
	store := newMockCellStore()
	rowKey := uuid.New()
	store.cells[cellKey(rowKey, "profile", 1)] = &cell.Cell{
		AddedID:    1,
		RowKey:     rowKey,
		ColumnName: "profile",
		RefKey:     1,
		Body:       json.RawMessage(`{"name":"test"}`),
		CreatedAt:  time.Now(),
	}

	server := setupTestServer(store, 64)

	req := httptest.NewRequest(http.MethodGet, "/v1/cells/"+rowKey.String()+"/profile/1", nil)
	w := httptest.NewRecorder()

	server.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status: got %d, want %d\nbody: %s", w.Code, http.StatusOK, w.Body.String())
	}
}

func TestGetCell_InvalidRowKey(t *testing.T) {
	store := newMockCellStore()
	server := setupTestServer(store, 64)

	req := httptest.NewRequest(http.MethodGet, "/v1/cells/not-a-uuid/profile/1", nil)
	w := httptest.NewRecorder()

	server.ServeHTTP(w, req)

	// Huma validates uuid format at the path param level
	if w.Code < 400 || w.Code >= 500 {
		t.Errorf("status: got %d, want 4xx", w.Code)
	}
}

func TestGetCell_InvalidRefKey(t *testing.T) {
	store := newMockCellStore()
	server := setupTestServer(store, 64)

	rowKey := uuid.New()
	req := httptest.NewRequest(http.MethodGet, "/v1/cells/"+rowKey.String()+"/profile/abc", nil)
	w := httptest.NewRecorder()

	server.ServeHTTP(w, req)

	if w.Code < 400 || w.Code >= 500 {
		t.Errorf("status: got %d, want 4xx", w.Code)
	}
}

func TestGetCell_NotFound(t *testing.T) {
	store := newMockCellStore()
	server := setupTestServer(store, 64)

	rowKey := uuid.New()
	req := httptest.NewRequest(http.MethodGet, "/v1/cells/"+rowKey.String()+"/profile/1", nil)
	w := httptest.NewRecorder()

	server.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status: got %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestGetCell_StoreError(t *testing.T) {
	store := newMockCellStore()
	store.getErr = errors.New("db error")
	server := setupTestServer(store, 64)

	rowKey := uuid.New()
	store.cells[cellKey(rowKey, "profile", 1)] = &cell.Cell{} // ensure shard routes
	req := httptest.NewRequest(http.MethodGet, "/v1/cells/"+rowKey.String()+"/profile/1", nil)
	w := httptest.NewRecorder()

	server.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status: got %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

// --- GetCellLatest Tests ---

func TestGetCellLatest_Success(t *testing.T) {
	store := newMockCellStore()
	rowKey := uuid.New()
	store.cells[cellKey(rowKey, "profile", 1)] = &cell.Cell{
		AddedID: 1, RowKey: rowKey, ColumnName: "profile", RefKey: 1,
		Body: json.RawMessage(`{"v":1}`), CreatedAt: time.Now(),
	}
	store.cells[cellKey(rowKey, "profile", 2)] = &cell.Cell{
		AddedID: 2, RowKey: rowKey, ColumnName: "profile", RefKey: 2,
		Body: json.RawMessage(`{"v":2}`), CreatedAt: time.Now(),
	}

	server := setupTestServer(store, 64)

	req := httptest.NewRequest(http.MethodGet, "/v1/cells/"+rowKey.String()+"/profile", nil)
	w := httptest.NewRecorder()

	server.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status: got %d, want %d\nbody: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp CellResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.RefKey != 2 {
		t.Errorf("RefKey: got %d, want 2 (latest)", resp.RefKey)
	}
}

func TestGetCellLatest_InvalidRowKey(t *testing.T) {
	store := newMockCellStore()
	server := setupTestServer(store, 64)

	req := httptest.NewRequest(http.MethodGet, "/v1/cells/invalid/profile", nil)
	w := httptest.NewRecorder()

	server.ServeHTTP(w, req)

	if w.Code < 400 || w.Code >= 500 {
		t.Errorf("status: got %d, want 4xx", w.Code)
	}
}

func TestGetCellLatest_NotFound(t *testing.T) {
	store := newMockCellStore()
	server := setupTestServer(store, 64)

	rowKey := uuid.New()
	req := httptest.NewRequest(http.MethodGet, "/v1/cells/"+rowKey.String()+"/profile", nil)
	w := httptest.NewRecorder()

	server.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status: got %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestGetCellLatest_StoreError(t *testing.T) {
	store := newMockCellStore()
	store.latestErr = errors.New("db error")
	server := setupTestServer(store, 64)

	rowKey := uuid.New()
	req := httptest.NewRequest(http.MethodGet, "/v1/cells/"+rowKey.String()+"/profile", nil)
	w := httptest.NewRecorder()

	server.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status: got %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

// --- GetRow Tests ---

func TestGetRow_Success(t *testing.T) {
	store := newMockCellStore()
	rowKey := uuid.New()
	store.rows[rowKey.String()] = []cell.Cell{
		{AddedID: 1, RowKey: rowKey, ColumnName: "profile", RefKey: 1, Body: json.RawMessage(`{}`), CreatedAt: time.Now()},
		{AddedID: 2, RowKey: rowKey, ColumnName: "settings", RefKey: 1, Body: json.RawMessage(`{}`), CreatedAt: time.Now()},
	}

	server := setupTestServer(store, 64)

	req := httptest.NewRequest(http.MethodGet, "/v1/cells/"+rowKey.String(), nil)
	w := httptest.NewRecorder()

	server.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status: got %d, want %d\nbody: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp RowResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.RowKey != rowKey {
		t.Errorf("RowKey: got %s", resp.RowKey)
	}
	if len(resp.Cells) != 2 {
		t.Errorf("Cells: got %d, want 2", len(resp.Cells))
	}
}

func TestGetRow_InvalidRowKey(t *testing.T) {
	store := newMockCellStore()
	server := setupTestServer(store, 64)

	req := httptest.NewRequest(http.MethodGet, "/v1/cells/not-a-uuid", nil)
	w := httptest.NewRecorder()

	server.ServeHTTP(w, req)

	if w.Code < 400 || w.Code >= 500 {
		t.Errorf("status: got %d, want 4xx", w.Code)
	}
}

func TestGetRow_Empty(t *testing.T) {
	store := newMockCellStore()
	server := setupTestServer(store, 64)

	rowKey := uuid.New()
	req := httptest.NewRequest(http.MethodGet, "/v1/cells/"+rowKey.String(), nil)
	w := httptest.NewRecorder()

	server.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status: got %d, want %d", w.Code, http.StatusOK)
	}
}

func TestGetRow_StoreError(t *testing.T) {
	store := newMockCellStore()
	store.rowErr = errors.New("db error")
	server := setupTestServer(store, 64)

	rowKey := uuid.New()
	req := httptest.NewRequest(http.MethodGet, "/v1/cells/"+rowKey.String(), nil)
	w := httptest.NewRecorder()

	server.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status: got %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

// --- Shard Routing Error Tests ---

func TestWriteCell_ShardRoutingError(t *testing.T) {
	// No stores registered
	server := NewServer(testLogger(), shard.NewRouter(), index.NewRegistry(), trigger.NewPluginRegistry(), nil, 64)

	body := map[string]any{
		"row_key":     uuid.New().String(),
		"column_name": "profile",
		"ref_key":     1,
		"body":        map[string]string{"name": "test"},
	}
	data, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/v1/cells", bytes.NewReader(data))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status: got %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

func TestGetCell_ShardRoutingError(t *testing.T) {
	server := NewServer(testLogger(), shard.NewRouter(), index.NewRegistry(), trigger.NewPluginRegistry(), nil, 64)

	rowKey := uuid.New()
	req := httptest.NewRequest(http.MethodGet, "/v1/cells/"+rowKey.String()+"/profile/1", nil)
	w := httptest.NewRecorder()

	server.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status: got %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

// --- NewCellHandler Tests ---

func TestNewCellHandler(t *testing.T) {
	router := shard.NewRouter()
	h := NewCellHandler(router, 64, index.NewRegistry(), nil, testLogger())
	if h == nil {
		t.Fatal("NewCellHandler returned nil")
	}
}
