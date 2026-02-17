package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
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
	cells           map[string]*cell.Cell
	rows            map[string][]cell.Cell
	partitionCells  []cell.Cell
	writeErr        error
	getErr          error
	latestErr       error
	rowErr          error
	partitionErr    error
	nextID          int64
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

func (m *mockCellStore) PartitionRead(ctx context.Context, partitionNumber int, readType int, cursor string, limit int) (*storage.Page, error) {
	if m.partitionErr != nil {
		return nil, m.partitionErr
	}
	
	// Simple mock: return all partition cells up to limit
	var cells []cell.Cell
	hasMore := false
	
	if len(m.partitionCells) > limit && limit > 0 {
		cells = m.partitionCells[:limit]
		hasMore = true
	} else {
		cells = m.partitionCells
	}
	
	var nextCursor string
	if hasMore && len(cells) > 0 {
		lastCell := cells[len(cells)-1]
		c := &storage.Cursor{AddedID: lastCell.AddedID}
		nextCursor, _ = c.Encode()
	}
	
	return &storage.Page{
		Cells:      cells,
		NextCursor: nextCursor,
		HasMore:    hasMore,
	}, nil
}

func (m *mockCellStore) ScanCells(ctx context.Context, columnName string, afterAddedID int64, limit int) ([]cell.Cell, error) {
	return nil, nil
}

func setupTestServer(store storage.CellStore, numShards int) http.Handler {
	r := shard.NewRouter()
	for i := 0; i < numShards; i++ {
		r.Register(shard.ID(i), store)
	}
	return NewServer(testLogger(), r, index.NewRegistry(), trigger.NewPluginRegistry(), nil, numShards, nil)
}

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
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

// --- PartitionRead Tests ---

func TestPartitionRead_Success(t *testing.T) {
	store := newMockCellStore()
	
	// Create test cells
	for i := 1; i <= 5; i++ {
		store.partitionCells = append(store.partitionCells, cell.Cell{
			AddedID:    int64(i),
			RowKey:     uuid.New(),
			ColumnName: "test",
			RefKey:     1,
			Body:       json.RawMessage(`{}`),
			CreatedAt:  time.Now(),
		})
	}
	
	server := setupTestServer(store, 64)

	req := httptest.NewRequest(http.MethodGet, "/v1/cells/partitionRead?partition_number=0&read_type=2&limit=3", nil)
	w := httptest.NewRecorder()

	server.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status: got %d, want 200\nbody: %s", w.Code, w.Body.String())
	}

	var resp PartitionReadOutput
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}

	if len(resp.Cells) != 3 {
		t.Errorf("cells: got %d, want 3", len(resp.Cells))
	}
	
	if !resp.HasMore {
		t.Error("expected HasMore to be true")
	}
	
	if resp.NextCursor == "" {
		t.Error("expected NextCursor to be non-empty")
	}
}

func TestPartitionRead_DefaultLimit(t *testing.T) {
	store := newMockCellStore()
	server := setupTestServer(store, 64)

	req := httptest.NewRequest(http.MethodGet, "/v1/cells/partitionRead?partition_number=0&read_type=2", nil)
	w := httptest.NewRecorder()

	server.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status: got %d, want 200\nbody: %s", w.Code, w.Body.String())
	}
}

func TestPartitionRead_InvalidType(t *testing.T) {
	store := newMockCellStore()
	server := setupTestServer(store, 64)

	req := httptest.NewRequest(http.MethodGet, "/v1/cells/partitionRead?partition_number=0&read_type=99", nil)
	w := httptest.NewRecorder()

	server.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status: got %d, want 400", w.Code)
	}
}

func TestPartitionRead_InvalidPartition(t *testing.T) {
	store := newMockCellStore()
	server := setupTestServer(store, 64)

	req := httptest.NewRequest(http.MethodGet, "/v1/cells/partitionRead?partition_number=999&read_type=2", nil)
	w := httptest.NewRecorder()

	server.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status: got %d, want 400", w.Code)
	}
}

func TestPartitionRead_CursorPagination(t *testing.T) {
	store := newMockCellStore()
	
	// Create 10 test cells
	for i := 1; i <= 10; i++ {
		store.partitionCells = append(store.partitionCells, cell.Cell{
			AddedID:    int64(i),
			RowKey:     uuid.New(),
			ColumnName: "test",
			RefKey:     1,
			Body:       json.RawMessage(`{}`),
			CreatedAt:  time.Now(),
		})
	}
	
	server := setupTestServer(store, 64)

	// First page
	req1 := httptest.NewRequest(http.MethodGet, "/v1/cells/partitionRead?partition_number=0&read_type=2&limit=5", nil)
	w1 := httptest.NewRecorder()
	server.ServeHTTP(w1, req1)

	if w1.Code != http.StatusOK {
		t.Fatalf("page 1: status: got %d, want 200", w1.Code)
	}

	var resp1 PartitionReadOutput
	if err := json.NewDecoder(w1.Body).Decode(&resp1); err != nil {
		t.Fatalf("decode page 1: %v", err)
	}

	if len(resp1.Cells) != 5 {
		t.Errorf("page 1 cells: got %d, want 5", len(resp1.Cells))
	}
	
	if !resp1.HasMore {
		t.Error("page 1: expected HasMore to be true")
	}

	// Second page using cursor
	req2 := httptest.NewRequest(http.MethodGet, "/v1/cells/partitionRead?partition_number=0&read_type=2&limit=5&cursor="+resp1.NextCursor, nil)
	w2 := httptest.NewRecorder()
	server.ServeHTTP(w2, req2)

	if w2.Code != http.StatusOK {
		t.Fatalf("page 2: status: got %d, want 200", w2.Code)
	}

	var resp2 PartitionReadOutput
	if err := json.NewDecoder(w2.Body).Decode(&resp2); err != nil {
		t.Fatalf("decode page 2: %v", err)
	}

	if len(resp2.Cells) != 5 {
		t.Errorf("page 2 cells: got %d, want 5", len(resp2.Cells))
	}
	
	if resp2.HasMore {
		t.Error("page 2: expected HasMore to be false")
	}
}
