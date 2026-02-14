package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/ryanbastic/go-mezzanine/internal/cell"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

var testPool *pgxpool.Pool

func TestMain(m *testing.M) {
	ctx := context.Background()

	ctr, err := postgres.Run(ctx, "postgres:16",
		postgres.WithDatabase("mezzanine"),
		postgres.WithUsername("postgres"),
		postgres.WithPassword("postgres"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(30*time.Second),
		),
	)
	if err != nil {
		panic(fmt.Sprintf("start postgres container: %v", err))
	}

	connStr, err := ctr.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		panic(fmt.Sprintf("get connection string: %v", err))
	}

	testPool, err = pgxpool.New(ctx, connStr)
	if err != nil {
		panic(fmt.Sprintf("create pool: %v", err))
	}

	code := m.Run()

	testPool.Close()
	_ = testcontainers.TerminateContainer(ctr)

	os.Exit(code)
}

var shardCounter int

// freshShard creates a new shard table with a unique ID and returns a store for it.
func freshShard(t *testing.T) *PostgresStore {
	t.Helper()
	shardCounter++
	shardID := 10000 + shardCounter
	ctx := context.Background()
	if err := RunMigrationsForPool(ctx, testPool, shardID, shardID); err != nil {
		t.Fatalf("run migrations for shard %d: %v", shardID, err)
	}
	return NewPostgresStore(testPool, shardID, 5*time.Second)
}

func TestWriteCell(t *testing.T) {
	store := freshShard(t)
	ctx := context.Background()

	rowKey := uuid.New()
	body := json.RawMessage(`{"name":"alice"}`)

	c, err := store.WriteCell(ctx, cell.WriteCellRequest{
		RowKey:     rowKey,
		ColumnName: "profile",
		RefKey:     1,
		Body:       body,
	})
	if err != nil {
		t.Fatalf("WriteCell: %v", err)
	}

	if c.AddedID == 0 {
		t.Error("expected non-zero AddedID")
	}
	if c.RowKey != rowKey {
		t.Errorf("RowKey = %v, want %v", c.RowKey, rowKey)
	}
	if c.ColumnName != "profile" {
		t.Errorf("ColumnName = %q, want %q", c.ColumnName, "profile")
	}
	if c.RefKey != 1 {
		t.Errorf("RefKey = %d, want 1", c.RefKey)
	}
	if c.CreatedAt.IsZero() {
		t.Error("expected non-zero CreatedAt")
	}
}

func TestWriteCell_DuplicateRefKey(t *testing.T) {
	store := freshShard(t)
	ctx := context.Background()

	rowKey := uuid.New()
	body := json.RawMessage(`{"v":1}`)

	req := cell.WriteCellRequest{
		RowKey:     rowKey,
		ColumnName: "col",
		RefKey:     1,
		Body:       body,
	}

	if _, err := store.WriteCell(ctx, req); err != nil {
		t.Fatalf("first WriteCell: %v", err)
	}

	_, err := store.WriteCell(ctx, req)
	if err == nil {
		t.Fatal("expected error on duplicate (row_key, column_name, ref_key)")
	}
}

func TestGetCell(t *testing.T) {
	store := freshShard(t)
	ctx := context.Background()

	rowKey := uuid.New()
	body := json.RawMessage(`{"x":42}`)

	written, err := store.WriteCell(ctx, cell.WriteCellRequest{
		RowKey:     rowKey,
		ColumnName: "data",
		RefKey:     10,
		Body:       body,
	})
	if err != nil {
		t.Fatalf("WriteCell: %v", err)
	}

	got, err := store.GetCell(ctx, cell.CellRef{
		RowKey:     rowKey,
		ColumnName: "data",
		RefKey:     10,
	})
	if err != nil {
		t.Fatalf("GetCell: %v", err)
	}

	if got.AddedID != written.AddedID {
		t.Errorf("AddedID = %d, want %d", got.AddedID, written.AddedID)
	}
	// Postgres normalizes JSONB whitespace, so compare parsed values.
	var gotBody, wantBody any
	if err := json.Unmarshal(got.Body, &gotBody); err != nil {
		t.Fatalf("unmarshal got body: %v", err)
	}
	if err := json.Unmarshal(body, &wantBody); err != nil {
		t.Fatalf("unmarshal want body: %v", err)
	}
	if fmt.Sprint(gotBody) != fmt.Sprint(wantBody) {
		t.Errorf("Body = %s, want %s", got.Body, body)
	}
}

func TestGetCell_NotFound(t *testing.T) {
	store := freshShard(t)
	ctx := context.Background()

	_, err := store.GetCell(ctx, cell.CellRef{
		RowKey:     uuid.New(),
		ColumnName: "missing",
		RefKey:     1,
	})
	if err != ErrCellNotFound {
		t.Fatalf("expected ErrCellNotFound, got %v", err)
	}
}

func TestGetCellLatest(t *testing.T) {
	store := freshShard(t)
	ctx := context.Background()

	rowKey := uuid.New()

	for i := int64(1); i <= 3; i++ {
		_, err := store.WriteCell(ctx, cell.WriteCellRequest{
			RowKey:     rowKey,
			ColumnName: "version",
			RefKey:     i,
			Body:       json.RawMessage(fmt.Sprintf(`{"v":%d}`, i)),
		})
		if err != nil {
			t.Fatalf("WriteCell ref_key=%d: %v", i, err)
		}
	}

	got, err := store.GetCellLatest(ctx, rowKey, "version")
	if err != nil {
		t.Fatalf("GetCellLatest: %v", err)
	}

	if got.RefKey != 3 {
		t.Errorf("RefKey = %d, want 3 (latest)", got.RefKey)
	}
}

func TestGetCellLatest_NotFound(t *testing.T) {
	store := freshShard(t)
	ctx := context.Background()

	_, err := store.GetCellLatest(ctx, uuid.New(), "nope")
	if err != ErrCellNotFound {
		t.Fatalf("expected ErrCellNotFound, got %v", err)
	}
}

func TestGetRow(t *testing.T) {
	store := freshShard(t)
	ctx := context.Background()

	rowKey := uuid.New()

	for _, req := range []cell.WriteCellRequest{
		{RowKey: rowKey, ColumnName: "email", RefKey: 1, Body: json.RawMessage(`{"v":"old@example.com"}`)},
		{RowKey: rowKey, ColumnName: "email", RefKey: 2, Body: json.RawMessage(`{"v":"new@example.com"}`)},
		{RowKey: rowKey, ColumnName: "name", RefKey: 1, Body: json.RawMessage(`{"v":"alice"}`)},
	} {
		if _, err := store.WriteCell(ctx, req); err != nil {
			t.Fatalf("WriteCell: %v", err)
		}
	}

	cells, err := store.GetRow(ctx, rowKey)
	if err != nil {
		t.Fatalf("GetRow: %v", err)
	}

	if len(cells) != 2 {
		t.Fatalf("len(cells) = %d, want 2 (one per column)", len(cells))
	}

	byCol := make(map[string]cell.Cell)
	for _, c := range cells {
		byCol[c.ColumnName] = c
	}

	if email, ok := byCol["email"]; !ok {
		t.Error("missing email column")
	} else if email.RefKey != 2 {
		t.Errorf("email RefKey = %d, want 2 (latest)", email.RefKey)
	}

	if _, ok := byCol["name"]; !ok {
		t.Error("missing name column")
	}
}

func TestGetRow_Empty(t *testing.T) {
	store := freshShard(t)
	ctx := context.Background()

	cells, err := store.GetRow(ctx, uuid.New())
	if err != nil {
		t.Fatalf("GetRow: %v", err)
	}
	if len(cells) != 0 {
		t.Errorf("expected 0 cells, got %d", len(cells))
	}
}

func TestScanCells(t *testing.T) {
	store := freshShard(t)
	ctx := context.Background()

	for i := int64(1); i <= 5; i++ {
		_, err := store.WriteCell(ctx, cell.WriteCellRequest{
			RowKey:     uuid.New(),
			ColumnName: "events",
			RefKey:     i,
			Body:       json.RawMessage(fmt.Sprintf(`{"seq":%d}`, i)),
		})
		if err != nil {
			t.Fatalf("WriteCell: %v", err)
		}
	}

	// Write a cell in a different column to confirm filtering
	_, err := store.WriteCell(ctx, cell.WriteCellRequest{
		RowKey:     uuid.New(),
		ColumnName: "other",
		RefKey:     1,
		Body:       json.RawMessage(`{}`),
	})
	if err != nil {
		t.Fatalf("WriteCell other: %v", err)
	}

	cells, err := store.ScanCells(ctx, "events", 0, 100)
	if err != nil {
		t.Fatalf("ScanCells: %v", err)
	}
	if len(cells) != 5 {
		t.Fatalf("len(cells) = %d, want 5", len(cells))
	}

	// Verify ordering by added_id ASC
	for i := 1; i < len(cells); i++ {
		if cells[i].AddedID <= cells[i-1].AddedID {
			t.Errorf("cells not in added_id ASC order: %d <= %d", cells[i].AddedID, cells[i-1].AddedID)
		}
	}

	// Scan with afterAddedID to skip first 2
	cells2, err := store.ScanCells(ctx, "events", cells[1].AddedID, 100)
	if err != nil {
		t.Fatalf("ScanCells after: %v", err)
	}
	if len(cells2) != 3 {
		t.Errorf("len(cells2) = %d, want 3", len(cells2))
	}

	// Scan with limit
	cells3, err := store.ScanCells(ctx, "events", 0, 2)
	if err != nil {
		t.Fatalf("ScanCells limit: %v", err)
	}
	if len(cells3) != 2 {
		t.Errorf("len(cells3) = %d, want 2", len(cells3))
	}
}

func TestPartitionRead_ByAddedID(t *testing.T) {
	store := freshShard(t)
	ctx := context.Background()

	var addedIDs []int64
	for i := int64(1); i <= 3; i++ {
		c, err := store.WriteCell(ctx, cell.WriteCellRequest{
			RowKey:     uuid.New(),
			ColumnName: "col",
			RefKey:     i,
			Body:       json.RawMessage(`{}`),
		})
		if err != nil {
			t.Fatalf("WriteCell: %v", err)
		}
		addedIDs = append(addedIDs, c.AddedID)
	}

	cells, err := store.PartitionRead(ctx, 0, PartitionReadTypeAddedID, 0, time.Time{}, 100)
	if err != nil {
		t.Fatalf("PartitionRead: %v", err)
	}
	if len(cells) != 3 {
		t.Fatalf("len(cells) = %d, want 3", len(cells))
	}

	cells2, err := store.PartitionRead(ctx, 0, PartitionReadTypeAddedID, addedIDs[0], time.Time{}, 100)
	if err != nil {
		t.Fatalf("PartitionRead after: %v", err)
	}
	if len(cells2) != 2 {
		t.Errorf("len(cells2) = %d, want 2", len(cells2))
	}
}

func TestPartitionRead_InvalidType(t *testing.T) {
	store := freshShard(t)
	ctx := context.Background()

	_, err := store.PartitionRead(ctx, 0, 999, 0, time.Time{}, 10)
	if err == nil {
		t.Fatal("expected error for invalid read type")
	}
}

func TestRunMigrationsForPool_MultipleShards(t *testing.T) {
	ctx := context.Background()

	shardCounter++
	base := 20000 + shardCounter*10
	if err := RunMigrationsForPool(ctx, testPool, base, base+3); err != nil {
		t.Fatalf("RunMigrationsForPool: %v", err)
	}

	for i := base; i <= base+3; i++ {
		store := NewPostgresStore(testPool, i, 5*time.Second)
		_, err := store.WriteCell(ctx, cell.WriteCellRequest{
			RowKey:     uuid.New(),
			ColumnName: "test",
			RefKey:     1,
			Body:       json.RawMessage(`{}`),
		})
		if err != nil {
			t.Errorf("WriteCell to shard %d: %v", i, err)
		}
	}
}

func TestRunMigrationsForPool_Idempotent(t *testing.T) {
	ctx := context.Background()

	shardCounter++
	shardID := 30000 + shardCounter
	if err := RunMigrationsForPool(ctx, testPool, shardID, shardID); err != nil {
		t.Fatalf("first migration: %v", err)
	}
	if err := RunMigrationsForPool(ctx, testPool, shardID, shardID); err != nil {
		t.Fatalf("second migration (idempotent): %v", err)
	}
}

func TestRunPluginMigration(t *testing.T) {
	ctx := context.Background()

	if err := RunPluginMigration(ctx, testPool); err != nil {
		t.Fatalf("RunPluginMigration: %v", err)
	}

	_, err := testPool.Exec(ctx, `
		INSERT INTO plugins (id, name, endpoint, subscribed_columns, status)
		VALUES ($1, $2, $3, $4, $5)
	`, uuid.New(), fmt.Sprintf("test-plugin-%d", time.Now().UnixNano()), "http://localhost:9090", []string{"col1"}, "active")
	if err != nil {
		t.Fatalf("insert into plugins: %v", err)
	}

	// Idempotent
	if err := RunPluginMigration(ctx, testPool); err != nil {
		t.Fatalf("second RunPluginMigration: %v", err)
	}
}
