package shard

import (
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/ryanbastic/go-mezzanine/internal/cell"
	"github.com/ryanbastic/go-mezzanine/internal/storage"
)

// mockCellStore implements storage.CellStore for testing.
type mockCellStore struct {
	id string // identifier for this mock
}

func (m *mockCellStore) WriteCell(ctx context.Context, req cell.WriteCellRequest) (*cell.Cell, error) {
	return &cell.Cell{
		AddedID:    1,
		RowKey:     req.RowKey,
		ColumnName: req.ColumnName,
		RefKey:     req.RefKey,
		Body:       req.Body,
		CreatedAt:  time.Now(),
	}, nil
}

func (m *mockCellStore) GetCell(ctx context.Context, ref cell.CellRef) (*cell.Cell, error) {
	return nil, storage.ErrCellNotFound
}

func (m *mockCellStore) GetCellLatest(ctx context.Context, rowKey uuid.UUID, columnName string) (*cell.Cell, error) {
	return nil, storage.ErrCellNotFound
}

func (m *mockCellStore) GetRow(ctx context.Context, rowKey uuid.UUID) ([]cell.Cell, error) {
	return nil, nil
}

func (m *mockCellStore) ScanCells(ctx context.Context, columnName string, afterAddedID int64, limit int) ([]cell.Cell, error) {
	return nil, nil
}

func TestNewRouter(t *testing.T) {
	r := NewRouter()
	if r == nil {
		t.Fatal("NewRouter returned nil")
	}
}

func TestRouter_RegisterAndStoreFor(t *testing.T) {
	r := NewRouter()
	store := &mockCellStore{id: "store-0"}
	r.Register(ID(0), store)

	got, err := r.StoreFor(ID(0))
	if err != nil {
		t.Fatalf("StoreFor: %v", err)
	}

	// Verify we got the same store back
	req := cell.WriteCellRequest{
		RowKey:     uuid.New(),
		ColumnName: "test",
		RefKey:     1,
		Body:       json.RawMessage(`{}`),
	}
	c, err := got.WriteCell(context.Background(), req)
	if err != nil {
		t.Fatalf("WriteCell: %v", err)
	}
	if c.RowKey != req.RowKey {
		t.Errorf("store returned wrong row key")
	}
}

func TestRouter_StoreFor_NotRegistered(t *testing.T) {
	r := NewRouter()

	_, err := r.StoreFor(ID(99))
	if err == nil {
		t.Fatal("expected error for unregistered shard")
	}
}

func TestRouter_MultipleShards(t *testing.T) {
	r := NewRouter()
	stores := make([]*mockCellStore, 4)
	for i := range stores {
		stores[i] = &mockCellStore{id: "store-" + string(rune('0'+i))}
		r.Register(ID(i), stores[i])
	}

	for i := 0; i < 4; i++ {
		_, err := r.StoreFor(ID(i))
		if err != nil {
			t.Errorf("shard %d: %v", i, err)
		}
	}
}

func TestRouter_OverwriteRegistration(t *testing.T) {
	r := NewRouter()
	store1 := &mockCellStore{id: "first"}
	store2 := &mockCellStore{id: "second"}

	r.Register(ID(0), store1)
	r.Register(ID(0), store2)

	got, err := r.StoreFor(ID(0))
	if err != nil {
		t.Fatalf("StoreFor: %v", err)
	}

	// Verify it returns the second store (last registered wins)
	if got != store2 {
		t.Error("expected second store to overwrite first")
	}
}

func TestRouter_ConcurrentAccess(t *testing.T) {
	r := NewRouter()
	for i := 0; i < 64; i++ {
		r.Register(ID(i), &mockCellStore{id: "store"})
	}

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(shardID int) {
			defer wg.Done()
			_, err := r.StoreFor(ID(shardID % 64))
			if err != nil {
				t.Errorf("concurrent StoreFor(%d): %v", shardID%64, err)
			}
		}(i)
	}
	wg.Wait()
}

func TestRouter_ConcurrentRegisterAndRead(t *testing.T) {
	r := NewRouter()

	var wg sync.WaitGroup

	// Writers
	for i := 0; i < 32; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			r.Register(ID(id), &mockCellStore{id: "store"})
		}(i)
	}

	// Readers (some will fail, that's OK)
	for i := 0; i < 32; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			r.StoreFor(ID(id))
		}(i)
	}

	wg.Wait()
}
