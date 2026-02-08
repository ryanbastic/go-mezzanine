package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/ryanbastic/go-mezzanine/internal/api"
	"github.com/ryanbastic/go-mezzanine/internal/cell"
	"github.com/ryanbastic/go-mezzanine/internal/index"
	"github.com/ryanbastic/go-mezzanine/internal/shard"
	"github.com/ryanbastic/go-mezzanine/internal/storage"
	"github.com/ryanbastic/go-mezzanine/internal/trigger"
)

// --- Mock CellStore (mirrors internal/api test mock) ---

type mockCellStore struct {
	cells  map[string]*cell.Cell
	rows   map[string][]cell.Cell
	nextID int64
}

func newMockCellStore() *mockCellStore {
	return &mockCellStore{
		cells: make(map[string]*cell.Cell),
		rows:  make(map[string][]cell.Cell),
	}
}

func mockCellKey(rowKey uuid.UUID, colName string, refKey int64) string {
	return fmt.Sprintf("%s:%s:%d", rowKey, colName, refKey)
}

func (m *mockCellStore) WriteCell(_ context.Context, req cell.WriteCellRequest) (*cell.Cell, error) {
	m.nextID++
	c := &cell.Cell{
		AddedID:    m.nextID,
		RowKey:     req.RowKey,
		ColumnName: req.ColumnName,
		RefKey:     req.RefKey,
		Body:       req.Body,
		CreatedAt:  time.Now(),
	}
	m.cells[mockCellKey(req.RowKey, req.ColumnName, req.RefKey)] = c
	m.rows[req.RowKey.String()] = append(m.rows[req.RowKey.String()], *c)
	return c, nil
}

func (m *mockCellStore) GetCell(_ context.Context, ref cell.CellRef) (*cell.Cell, error) {
	c, ok := m.cells[mockCellKey(ref.RowKey, ref.ColumnName, ref.RefKey)]
	if !ok {
		return nil, storage.ErrCellNotFound
	}
	return c, nil
}

func (m *mockCellStore) GetCellLatest(_ context.Context, rowKey uuid.UUID, columnName string) (*cell.Cell, error) {
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

func (m *mockCellStore) GetRow(_ context.Context, rowKey uuid.UUID) ([]cell.Cell, error) {
	return m.rows[rowKey.String()], nil
}

func (m *mockCellStore) PartitionRead(context.Context, int, int, int64, time.Time, int) ([]cell.Cell, error) {
	return nil, nil
}

func (m *mockCellStore) ScanCells(context.Context, string, int64, int) ([]cell.Cell, error) {
	return nil, nil
}

// testServerWithCells returns a server with mock cell stores (no index registry).
// Use this for write/read cell tests where IndexCell would hit a nil pool.
func testServerWithCells(t *testing.T) *httptest.Server {
	t.Helper()

	store := newMockCellStore()
	router := shard.NewRouter()
	for i := range 64 {
		router.Register(shard.ID(i), store)
	}

	logger := slog.New(slog.DiscardHandler)
	handler := api.NewServer(logger, router, index.NewRegistry(), trigger.NewPluginRegistry(), nil, 64, nil)
	return httptest.NewServer(handler)
}

// testServerWithIndex returns a server with the user_by_email index registered.
// Cell stores are wired up but the index stores use nil pools (query routing only).
func testServerWithIndex(t *testing.T) *httptest.Server {
	t.Helper()

	store := newMockCellStore()
	router := shard.NewRouter()
	for i := range 64 {
		router.Register(shard.ID(i), store)
	}

	registry := index.NewRegistry()
	registry.Register(nil, index.Definition{
		Name:          "user_by_email",
		SourceColumn:  "profile",
		ShardKeyField: "email",
		Fields:        []string{"email", "display_name"},
		UniqueFields:  []string{"email"},
	}, 64)

	logger := slog.New(slog.DiscardHandler)
	handler := api.NewServer(logger, router, registry, trigger.NewPluginRegistry(), nil, 64, nil)
	return httptest.NewServer(handler)
}

func TestUserByEmail_WriteProfileCell(t *testing.T) {
	srv := testServerWithCells(t)
	defer srv.Close()

	rowKey := uuid.New().String()

	body := map[string]any{
		"row_key":     rowKey,
		"column_name": "profile",
		"ref_key":     1,
		"body": map[string]any{
			"email":        "ryan@bastic.net",
			"display_name": "Ryan Bastic",
		},
	}
	data, _ := json.Marshal(body)

	resp, err := http.Post(srv.URL+"/v1/cells", "application/json", bytes.NewReader(data))
	if err != nil {
		t.Fatalf("POST /v1/cells: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("status: got %d, want %d\nbody: %s", resp.StatusCode, http.StatusCreated, string(b))
	}

	var cellResp map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&cellResp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if cellResp["row_key"] != rowKey {
		t.Errorf("row_key: got %v, want %s", cellResp["row_key"], rowKey)
	}
	if cellResp["column_name"] != "profile" {
		t.Errorf("column_name: got %v", cellResp["column_name"])
	}

	// Verify the body payload round-trips correctly.
	bodyMap, ok := cellResp["body"].(map[string]any)
	if !ok {
		t.Fatalf("body: got %T, want map", cellResp["body"])
	}
	if bodyMap["email"] != "ryan@bastic.net" {
		t.Errorf("email: got %v", bodyMap["email"])
	}
	if bodyMap["display_name"] != "Ryan Bastic" {
		t.Errorf("display_name: got %v", bodyMap["display_name"])
	}
}

func TestUserByEmail_GetRow(t *testing.T) {
	srv := testServerWithCells(t)
	defer srv.Close()

	rowKey := uuid.New().String()

	// Write the cell first.
	body := map[string]any{
		"row_key":     rowKey,
		"column_name": "profile",
		"ref_key":     1,
		"body": map[string]any{
			"email":        "ryan@bastic.net",
			"display_name": "Ryan Bastic",
		},
	}
	data, _ := json.Marshal(body)

	writeResp, err := http.Post(srv.URL+"/v1/cells", "application/json", bytes.NewReader(data))
	if err != nil {
		t.Fatalf("POST /v1/cells: %v", err)
	}
	writeResp.Body.Close()
	if writeResp.StatusCode != http.StatusCreated {
		t.Fatalf("write status: %d", writeResp.StatusCode)
	}

	// Read back the full row.
	getResp, err := http.Get(srv.URL + "/v1/cells/" + rowKey)
	if err != nil {
		t.Fatalf("GET /v1/cells/%s: %v", rowKey, err)
	}
	defer getResp.Body.Close()

	if getResp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(getResp.Body)
		t.Fatalf("status: got %d, want %d\nbody: %s", getResp.StatusCode, http.StatusOK, string(b))
	}

	var rowResp map[string]any
	if err := json.NewDecoder(getResp.Body).Decode(&rowResp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if rowResp["row_key"] != rowKey {
		t.Errorf("row_key: got %v", rowResp["row_key"])
	}
}

func TestUserByEmail_QueryIndexRoute(t *testing.T) {
	srv := testServerWithIndex(t)
	defer srv.Close()

	// Query the user_by_email index — the route should resolve (not 404)
	// because the definition is registered.
	resp, err := http.Get(fmt.Sprintf("%s/v1/index/user_by_email/ryan@bastic.net", srv.URL))
	if err != nil {
		t.Fatalf("GET /v1/index/user_by_email: %v", err)
	}
	defer resp.Body.Close()

	// The store has a nil pool, so QueryByShardKey returns 500,
	// but critically it does NOT return 404 — the index route resolved.
	if resp.StatusCode == http.StatusNotFound {
		t.Error("expected index route to resolve (not 404)")
	}
}

func TestUserByEmail_QueryNonexistentIndex(t *testing.T) {
	srv := testServerWithIndex(t)
	defer srv.Close()

	resp, err := http.Get(fmt.Sprintf("%s/v1/index/nonexistent/ryan@bastic.net", srv.URL))
	if err != nil {
		t.Fatalf("GET /v1/index/nonexistent: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("status: got %d, want %d", resp.StatusCode, http.StatusNotFound)
	}
}
