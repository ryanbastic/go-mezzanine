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
	"github.com/ryanbastic/go-mezzanine/internal/shard"
	"github.com/ryanbastic/go-mezzanine/internal/storage"
)

// --- Mock CellStore ---

type mockCellStore struct {
	cells      map[string]*cell.Cell // keyed by "rowKey:colName:refKey"
	rows       map[string][]cell.Cell
	writeErr   error
	getErr     error
	latestErr  error
	rowErr     error
	nextID     int64
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
	// Find cell with highest ref_key for this row+column
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

func (m *mockCellStore) ScanCells(ctx context.Context, columnName string, afterAddedID int64, limit int) ([]cell.Cell, error) {
	return nil, nil
}

func setupRouter(store storage.CellStore, numShards int) *shard.Router {
	r := shard.NewRouter()
	for i := 0; i < numShards; i++ {
		r.Register(shard.ID(i), store)
	}
	return r
}

func chiContext(r *http.Request, params map[string]string) *http.Request {
	rctx := chi.NewRouteContext()
	for k, v := range params {
		rctx.URLParams.Add(k, v)
	}
	return r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
}

// --- WriteCell Tests ---

func TestWriteCell_Success(t *testing.T) {
	store := newMockCellStore()
	router := setupRouter(store, 64)
	handler := NewCellHandler(router, 64, testLogger())

	rowKey := uuid.New()
	body := map[string]any{
		"row_key":     rowKey.String(),
		"column_name": "profile",
		"ref_key":     1,
		"body":        map[string]string{"name": "test"},
	}
	data, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/v1/cells", bytes.NewReader(data))
	w := httptest.NewRecorder()

	handler.WriteCell(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("status: got %d, want %d", w.Code, http.StatusCreated)
	}

	var resp cell.Cell
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
	router := setupRouter(store, 64)
	handler := NewCellHandler(router, 64, testLogger())

	req := httptest.NewRequest(http.MethodPost, "/v1/cells", bytes.NewReader([]byte("invalid json")))
	w := httptest.NewRecorder()

	handler.WriteCell(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status: got %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestWriteCell_MissingRowKey(t *testing.T) {
	store := newMockCellStore()
	router := setupRouter(store, 64)
	handler := NewCellHandler(router, 64, testLogger())

	body := map[string]any{
		"column_name": "profile",
		"ref_key":     1,
		"body":        map[string]string{"name": "test"},
	}
	data, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/v1/cells", bytes.NewReader(data))
	w := httptest.NewRecorder()

	handler.WriteCell(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status: got %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestWriteCell_MissingColumnName(t *testing.T) {
	store := newMockCellStore()
	router := setupRouter(store, 64)
	handler := NewCellHandler(router, 64, testLogger())

	body := map[string]any{
		"row_key": uuid.New().String(),
		"ref_key": 1,
		"body":    map[string]string{"name": "test"},
	}
	data, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/v1/cells", bytes.NewReader(data))
	w := httptest.NewRecorder()

	handler.WriteCell(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status: got %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestWriteCell_MissingBody(t *testing.T) {
	store := newMockCellStore()
	router := setupRouter(store, 64)
	handler := NewCellHandler(router, 64, testLogger())

	body := map[string]any{
		"row_key":     uuid.New().String(),
		"column_name": "profile",
		"ref_key":     1,
	}
	data, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/v1/cells", bytes.NewReader(data))
	w := httptest.NewRecorder()

	handler.WriteCell(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status: got %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestWriteCell_StoreError(t *testing.T) {
	store := newMockCellStore()
	store.writeErr = errors.New("db error")
	router := setupRouter(store, 64)
	handler := NewCellHandler(router, 64, testLogger())

	body := map[string]any{
		"row_key":     uuid.New().String(),
		"column_name": "profile",
		"ref_key":     1,
		"body":        map[string]string{"name": "test"},
	}
	data, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/v1/cells", bytes.NewReader(data))
	w := httptest.NewRecorder()

	handler.WriteCell(w, req)

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

	router := setupRouter(store, 64)
	handler := NewCellHandler(router, 64, testLogger())

	req := httptest.NewRequest(http.MethodGet, "/v1/cells/"+rowKey.String()+"/profile/1", nil)
	req = chiContext(req, map[string]string{
		"row_key":     rowKey.String(),
		"column_name": "profile",
		"ref_key":     "1",
	})
	w := httptest.NewRecorder()

	handler.GetCell(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status: got %d, want %d", w.Code, http.StatusOK)
	}
}

func TestGetCell_InvalidRowKey(t *testing.T) {
	store := newMockCellStore()
	router := setupRouter(store, 64)
	handler := NewCellHandler(router, 64, testLogger())

	req := httptest.NewRequest(http.MethodGet, "/v1/cells/not-a-uuid/profile/1", nil)
	req = chiContext(req, map[string]string{
		"row_key":     "not-a-uuid",
		"column_name": "profile",
		"ref_key":     "1",
	})
	w := httptest.NewRecorder()

	handler.GetCell(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status: got %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestGetCell_InvalidRefKey(t *testing.T) {
	store := newMockCellStore()
	router := setupRouter(store, 64)
	handler := NewCellHandler(router, 64, testLogger())

	rowKey := uuid.New()
	req := httptest.NewRequest(http.MethodGet, "/v1/cells/"+rowKey.String()+"/profile/abc", nil)
	req = chiContext(req, map[string]string{
		"row_key":     rowKey.String(),
		"column_name": "profile",
		"ref_key":     "abc",
	})
	w := httptest.NewRecorder()

	handler.GetCell(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status: got %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestGetCell_NotFound(t *testing.T) {
	store := newMockCellStore()
	router := setupRouter(store, 64)
	handler := NewCellHandler(router, 64, testLogger())

	rowKey := uuid.New()
	req := httptest.NewRequest(http.MethodGet, "/v1/cells/"+rowKey.String()+"/profile/1", nil)
	req = chiContext(req, map[string]string{
		"row_key":     rowKey.String(),
		"column_name": "profile",
		"ref_key":     "1",
	})
	w := httptest.NewRecorder()

	handler.GetCell(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status: got %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestGetCell_StoreError(t *testing.T) {
	store := newMockCellStore()
	store.getErr = errors.New("db error")
	router := setupRouter(store, 64)
	handler := NewCellHandler(router, 64, testLogger())

	rowKey := uuid.New()
	req := httptest.NewRequest(http.MethodGet, "/v1/cells/"+rowKey.String()+"/profile/1", nil)
	req = chiContext(req, map[string]string{
		"row_key":     rowKey.String(),
		"column_name": "profile",
		"ref_key":     "1",
	})
	w := httptest.NewRecorder()

	handler.GetCell(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status: got %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

// --- GetCellLatest Tests ---

func TestGetCellLatest_Success(t *testing.T) {
	store := newMockCellStore()
	rowKey := uuid.New()
	store.cells[cellKey(rowKey, "profile", 1)] = &cell.Cell{
		AddedID:    1,
		RowKey:     rowKey,
		ColumnName: "profile",
		RefKey:     1,
		Body:       json.RawMessage(`{"v":1}`),
		CreatedAt:  time.Now(),
	}
	store.cells[cellKey(rowKey, "profile", 2)] = &cell.Cell{
		AddedID:    2,
		RowKey:     rowKey,
		ColumnName: "profile",
		RefKey:     2,
		Body:       json.RawMessage(`{"v":2}`),
		CreatedAt:  time.Now(),
	}

	router := setupRouter(store, 64)
	handler := NewCellHandler(router, 64, testLogger())

	req := httptest.NewRequest(http.MethodGet, "/v1/cells/"+rowKey.String()+"/profile", nil)
	req = chiContext(req, map[string]string{
		"row_key":     rowKey.String(),
		"column_name": "profile",
	})
	w := httptest.NewRecorder()

	handler.GetCellLatest(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status: got %d, want %d", w.Code, http.StatusOK)
	}

	var resp cell.Cell
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.RefKey != 2 {
		t.Errorf("RefKey: got %d, want 2 (latest)", resp.RefKey)
	}
}

func TestGetCellLatest_InvalidRowKey(t *testing.T) {
	store := newMockCellStore()
	router := setupRouter(store, 64)
	handler := NewCellHandler(router, 64, testLogger())

	req := httptest.NewRequest(http.MethodGet, "/v1/cells/invalid/profile", nil)
	req = chiContext(req, map[string]string{
		"row_key":     "invalid",
		"column_name": "profile",
	})
	w := httptest.NewRecorder()

	handler.GetCellLatest(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status: got %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestGetCellLatest_NotFound(t *testing.T) {
	store := newMockCellStore()
	router := setupRouter(store, 64)
	handler := NewCellHandler(router, 64, testLogger())

	rowKey := uuid.New()
	req := httptest.NewRequest(http.MethodGet, "/v1/cells/"+rowKey.String()+"/profile", nil)
	req = chiContext(req, map[string]string{
		"row_key":     rowKey.String(),
		"column_name": "profile",
	})
	w := httptest.NewRecorder()

	handler.GetCellLatest(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status: got %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestGetCellLatest_StoreError(t *testing.T) {
	store := newMockCellStore()
	store.latestErr = errors.New("db error")
	router := setupRouter(store, 64)
	handler := NewCellHandler(router, 64, testLogger())

	rowKey := uuid.New()
	req := httptest.NewRequest(http.MethodGet, "/v1/cells/"+rowKey.String()+"/profile", nil)
	req = chiContext(req, map[string]string{
		"row_key":     rowKey.String(),
		"column_name": "profile",
	})
	w := httptest.NewRecorder()

	handler.GetCellLatest(w, req)

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

	router := setupRouter(store, 64)
	handler := NewCellHandler(router, 64, testLogger())

	req := httptest.NewRequest(http.MethodGet, "/v1/cells/"+rowKey.String(), nil)
	req = chiContext(req, map[string]string{
		"row_key": rowKey.String(),
	})
	w := httptest.NewRecorder()

	handler.GetRow(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status: got %d, want %d", w.Code, http.StatusOK)
	}

	var resp struct {
		RowKey uuid.UUID   `json:"row_key"`
		Cells  []cell.Cell `json:"cells"`
	}
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
	router := setupRouter(store, 64)
	handler := NewCellHandler(router, 64, testLogger())

	req := httptest.NewRequest(http.MethodGet, "/v1/cells/not-a-uuid", nil)
	req = chiContext(req, map[string]string{
		"row_key": "not-a-uuid",
	})
	w := httptest.NewRecorder()

	handler.GetRow(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status: got %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestGetRow_Empty(t *testing.T) {
	store := newMockCellStore()
	router := setupRouter(store, 64)
	handler := NewCellHandler(router, 64, testLogger())

	rowKey := uuid.New()
	req := httptest.NewRequest(http.MethodGet, "/v1/cells/"+rowKey.String(), nil)
	req = chiContext(req, map[string]string{
		"row_key": rowKey.String(),
	})
	w := httptest.NewRecorder()

	handler.GetRow(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status: got %d, want %d", w.Code, http.StatusOK)
	}
}

func TestGetRow_StoreError(t *testing.T) {
	store := newMockCellStore()
	store.rowErr = errors.New("db error")
	router := setupRouter(store, 64)
	handler := NewCellHandler(router, 64, testLogger())

	rowKey := uuid.New()
	req := httptest.NewRequest(http.MethodGet, "/v1/cells/"+rowKey.String(), nil)
	req = chiContext(req, map[string]string{
		"row_key": rowKey.String(),
	})
	w := httptest.NewRecorder()

	handler.GetRow(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status: got %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

// --- NewCellHandler Tests ---

func TestNewCellHandler(t *testing.T) {
	router := shard.NewRouter()
	h := NewCellHandler(router, 64, testLogger())
	if h == nil {
		t.Fatal("NewCellHandler returned nil")
	}
}

// --- Shard Routing Error Tests ---

func TestWriteCell_ShardRoutingError(t *testing.T) {
	// Router with no stores registered
	router := shard.NewRouter()
	handler := NewCellHandler(router, 64, testLogger())

	body := map[string]any{
		"row_key":     uuid.New().String(),
		"column_name": "profile",
		"ref_key":     1,
		"body":        map[string]string{"name": "test"},
	}
	data, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/v1/cells", bytes.NewReader(data))
	w := httptest.NewRecorder()

	handler.WriteCell(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status: got %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

func TestGetCell_ShardRoutingError(t *testing.T) {
	router := shard.NewRouter()
	handler := NewCellHandler(router, 64, testLogger())

	rowKey := uuid.New()
	req := httptest.NewRequest(http.MethodGet, "/v1/cells/"+rowKey.String()+"/profile/1", nil)
	req = chiContext(req, map[string]string{
		"row_key":     rowKey.String(),
		"column_name": "profile",
		"ref_key":     "1",
	})
	w := httptest.NewRecorder()

	handler.GetCell(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status: got %d, want %d", w.Code, http.StatusInternalServerError)
	}
}
