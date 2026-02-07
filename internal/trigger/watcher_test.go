package trigger

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/ryanbastic/go-mezzanine/internal/cell"
	"github.com/ryanbastic/go-mezzanine/internal/shard"
	"github.com/ryanbastic/go-mezzanine/internal/storage"
)

// --- Mock CellStore ---

type mockCellStore struct {
	mu    sync.Mutex
	cells []cell.Cell
}

func newMockStore(cells ...cell.Cell) *mockCellStore {
	return &mockCellStore{cells: cells}
}

func (m *mockCellStore) WriteCell(ctx context.Context, req cell.WriteCellRequest) (*cell.Cell, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	c := cell.Cell{
		AddedID:    int64(len(m.cells) + 1),
		RowKey:     req.RowKey,
		ColumnName: req.ColumnName,
		RefKey:     req.RefKey,
		Body:       req.Body,
		CreatedAt:  time.Now(),
	}
	m.cells = append(m.cells, c)
	return &c, nil
}

func (m *mockCellStore) GetCell(ctx context.Context, ref cell.CellRef) (*cell.Cell, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, c := range m.cells {
		if c.RowKey == ref.RowKey && c.ColumnName == ref.ColumnName && c.RefKey == ref.RefKey {
			return &c, nil
		}
	}
	return nil, storage.ErrCellNotFound
}

func (m *mockCellStore) GetCellLatest(ctx context.Context, rowKey uuid.UUID, columnName string) (*cell.Cell, error) {
	return nil, storage.ErrCellNotFound
}

func (m *mockCellStore) GetRow(ctx context.Context, rowKey uuid.UUID) ([]cell.Cell, error) {
	return nil, nil
}

func (m *mockCellStore) PartitionRead(ctx context.Context, partitionNumber int, readType int, addedID int64, createdAfter time.Time, limit int) ([]cell.Cell, error) {
	return nil, nil
}

func (m *mockCellStore) ScanCells(ctx context.Context, columnName string, afterAddedID int64, limit int) ([]cell.Cell, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var result []cell.Cell
	for _, c := range m.cells {
		if c.ColumnName == columnName && c.AddedID > afterAddedID {
			result = append(result, c)
			if len(result) >= limit {
				break
			}
		}
	}
	return result, nil
}

// --- Mock Checkpoint ---

type mockCheckpoint struct {
	mu     sync.Mutex
	values map[string]int64 // "shardID:column" -> lastAddedID
}

func newMockCheckpoint() *mockCheckpoint {
	return &mockCheckpoint{values: make(map[string]int64)}
}

func (m *mockCheckpoint) key(shardID shard.ID, columnName string) string {
	return string(rune(shardID)) + ":" + columnName
}

func (m *mockCheckpoint) Load(ctx context.Context, shardID shard.ID, columnName string) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.values[m.key(shardID, columnName)], nil
}

func (m *mockCheckpoint) Save(ctx context.Context, shardID shard.ID, columnName string, addedID int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.values[m.key(shardID, columnName)] = addedID
	return nil
}

func testLogger() *slog.Logger {
	return slog.New(slog.DiscardHandler)
}

func TestNewWatcher(t *testing.T) {
	reg := NewRegistry()
	cp := newMockCheckpoint()
	stores := map[shard.ID]storage.CellStore{}
	w := NewWatcher(reg, cp, stores, 1, 10*time.Millisecond, 10, testLogger())
	if w == nil {
		t.Fatal("NewWatcher returned nil")
	}
}

func TestWatcher_NoHandlers_ReturnsImmediately(t *testing.T) {
	reg := NewRegistry()
	cp := newMockCheckpoint()
	stores := map[shard.ID]storage.CellStore{}
	w := NewWatcher(reg, cp, stores, 1, 10*time.Millisecond, 10, testLogger())

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start should return immediately since no handlers are registered
	done := make(chan struct{})
	go func() {
		w.Start(ctx)
		close(done)
	}()

	select {
	case <-done:
		// Good, it returned
	case <-time.After(time.Second):
		t.Error("Start should return immediately with no handlers")
	}
}

func TestWatcher_ProcessesCells(t *testing.T) {
	reg := NewRegistry()

	var mu sync.Mutex
	var processed []int64

	reg.Register("events", func(ctx context.Context, c cell.Cell) error {
		mu.Lock()
		processed = append(processed, c.AddedID)
		mu.Unlock()
		return nil
	})

	store := newMockStore(
		cell.Cell{AddedID: 1, ColumnName: "events", RowKey: uuid.New(), Body: json.RawMessage(`{}`)},
		cell.Cell{AddedID: 2, ColumnName: "events", RowKey: uuid.New(), Body: json.RawMessage(`{}`)},
		cell.Cell{AddedID: 3, ColumnName: "events", RowKey: uuid.New(), Body: json.RawMessage(`{}`)},
	)

	cp := newMockCheckpoint()
	stores := map[shard.ID]storage.CellStore{shard.ID(0): store}

	w := NewWatcher(reg, cp, stores, 1, 10*time.Millisecond, 100, testLogger())

	ctx, cancel := context.WithCancel(context.Background())
	w.Start(ctx)

	// Wait for processing
	time.Sleep(100 * time.Millisecond)
	cancel()

	mu.Lock()
	defer mu.Unlock()

	if len(processed) != 3 {
		t.Errorf("expected 3 processed cells, got %d", len(processed))
	}
}

func TestWatcher_CheckpointSaved(t *testing.T) {
	reg := NewRegistry()
	reg.Register("events", func(ctx context.Context, c cell.Cell) error {
		return nil
	})

	store := newMockStore(
		cell.Cell{AddedID: 1, ColumnName: "events", RowKey: uuid.New(), Body: json.RawMessage(`{}`)},
		cell.Cell{AddedID: 2, ColumnName: "events", RowKey: uuid.New(), Body: json.RawMessage(`{}`)},
	)

	cp := newMockCheckpoint()
	stores := map[shard.ID]storage.CellStore{shard.ID(0): store}

	w := NewWatcher(reg, cp, stores, 1, 10*time.Millisecond, 100, testLogger())

	ctx, cancel := context.WithCancel(context.Background())
	w.Start(ctx)

	time.Sleep(100 * time.Millisecond)
	cancel()

	// Allow goroutine to save final checkpoint
	time.Sleep(50 * time.Millisecond)

	cp.mu.Lock()
	defer cp.mu.Unlock()

	// Check that checkpoint was saved
	key := cp.key(shard.ID(0), "events")
	if cp.values[key] < 2 {
		t.Errorf("expected checkpoint >= 2, got %d", cp.values[key])
	}
}

func TestWatcher_HandlerError_StopsBatch(t *testing.T) {
	reg := NewRegistry()

	var mu sync.Mutex
	var processed []int64

	reg.Register("events", func(ctx context.Context, c cell.Cell) error {
		mu.Lock()
		processed = append(processed, c.AddedID)
		mu.Unlock()
		if c.AddedID == 2 {
			return errors.New("handler error")
		}
		return nil
	})

	store := newMockStore(
		cell.Cell{AddedID: 1, ColumnName: "events", RowKey: uuid.New(), Body: json.RawMessage(`{}`)},
		cell.Cell{AddedID: 2, ColumnName: "events", RowKey: uuid.New(), Body: json.RawMessage(`{}`)},
		cell.Cell{AddedID: 3, ColumnName: "events", RowKey: uuid.New(), Body: json.RawMessage(`{}`)},
	)

	cp := newMockCheckpoint()
	stores := map[shard.ID]storage.CellStore{shard.ID(0): store}

	w := NewWatcher(reg, cp, stores, 1, 10*time.Millisecond, 100, testLogger())

	ctx, cancel := context.WithCancel(context.Background())
	w.Start(ctx)

	// Let it process first batch and retry
	time.Sleep(100 * time.Millisecond)
	cancel()
	time.Sleep(50 * time.Millisecond)

	cp.mu.Lock()
	cpVal := cp.values[cp.key(shard.ID(0), "events")]
	cp.mu.Unlock()

	// Checkpoint should be stuck at 1 (before the failing cell)
	// because processBatch returns lastID before the error
	if cpVal > 1 {
		t.Errorf("expected checkpoint <= 1 (before failing cell), got %d", cpVal)
	}
}

func TestWatcher_IgnoresOtherColumns(t *testing.T) {
	reg := NewRegistry()

	var mu sync.Mutex
	var processed []int64

	reg.Register("events", func(ctx context.Context, c cell.Cell) error {
		mu.Lock()
		processed = append(processed, c.AddedID)
		mu.Unlock()
		return nil
	})

	store := newMockStore(
		cell.Cell{AddedID: 1, ColumnName: "events", RowKey: uuid.New(), Body: json.RawMessage(`{}`)},
		cell.Cell{AddedID: 2, ColumnName: "other", RowKey: uuid.New(), Body: json.RawMessage(`{}`)},
		cell.Cell{AddedID: 3, ColumnName: "events", RowKey: uuid.New(), Body: json.RawMessage(`{}`)},
	)

	cp := newMockCheckpoint()
	stores := map[shard.ID]storage.CellStore{shard.ID(0): store}

	w := NewWatcher(reg, cp, stores, 1, 10*time.Millisecond, 100, testLogger())

	ctx, cancel := context.WithCancel(context.Background())
	w.Start(ctx)

	time.Sleep(100 * time.Millisecond)
	cancel()

	mu.Lock()
	defer mu.Unlock()

	// Should only process cells with ColumnName "events" — the mock ScanCells
	// filters by column, so only AddedID 1 and 3 should be processed
	if len(processed) != 2 {
		t.Errorf("expected 2 processed cells (events only), got %d: %v", len(processed), processed)
	}
}

func TestProcessBatch_EmptyCells(t *testing.T) {
	reg := NewRegistry()
	reg.Register("events", func(ctx context.Context, c cell.Cell) error { return nil })

	store := newMockStore() // empty store
	cp := newMockCheckpoint()
	stores := map[shard.ID]storage.CellStore{shard.ID(0): store}

	w := NewWatcher(reg, cp, stores, 1, time.Hour, 100, testLogger())

	lastID, err := w.processBatch(context.Background(), store, shard.ID(0), "events", 0)
	if err != nil {
		t.Fatalf("processBatch: %v", err)
	}
	if lastID != 0 {
		t.Errorf("expected lastID 0, got %d", lastID)
	}
}

func TestProcessBatch_AllSuccess(t *testing.T) {
	reg := NewRegistry()

	callCount := 0
	reg.Register("events", func(ctx context.Context, c cell.Cell) error {
		callCount++
		return nil
	})

	store := newMockStore(
		cell.Cell{AddedID: 10, ColumnName: "events", RowKey: uuid.New(), Body: json.RawMessage(`{}`)},
		cell.Cell{AddedID: 20, ColumnName: "events", RowKey: uuid.New(), Body: json.RawMessage(`{}`)},
	)

	cp := newMockCheckpoint()
	stores := map[shard.ID]storage.CellStore{shard.ID(0): store}

	w := NewWatcher(reg, cp, stores, 1, time.Hour, 100, testLogger())

	lastID, err := w.processBatch(context.Background(), store, shard.ID(0), "events", 0)
	if err != nil {
		t.Fatalf("processBatch: %v", err)
	}
	if lastID != 20 {
		t.Errorf("expected lastID 20, got %d", lastID)
	}
	if callCount != 2 {
		t.Errorf("expected 2 handler calls, got %d", callCount)
	}
}

func TestProcessBatch_MultipleHandlers(t *testing.T) {
	reg := NewRegistry()

	var order []string
	reg.Register("events", func(ctx context.Context, c cell.Cell) error {
		order = append(order, "h1")
		return nil
	})
	reg.Register("events", func(ctx context.Context, c cell.Cell) error {
		order = append(order, "h2")
		return nil
	})

	store := newMockStore(
		cell.Cell{AddedID: 1, ColumnName: "events", RowKey: uuid.New(), Body: json.RawMessage(`{}`)},
	)

	cp := newMockCheckpoint()
	stores := map[shard.ID]storage.CellStore{shard.ID(0): store}

	w := NewWatcher(reg, cp, stores, 1, time.Hour, 100, testLogger())

	lastID, err := w.processBatch(context.Background(), store, shard.ID(0), "events", 0)
	if err != nil {
		t.Fatalf("processBatch: %v", err)
	}
	if lastID != 1 {
		t.Errorf("expected lastID 1, got %d", lastID)
	}
	if len(order) != 2 || order[0] != "h1" || order[1] != "h2" {
		t.Errorf("unexpected handler order: %v", order)
	}
}
