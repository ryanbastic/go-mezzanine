package trigger

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"github.com/google/uuid"
)

// mockPluginStore is an in-memory implementation of PluginStore for testing.
type mockPluginStore struct {
	mu      sync.Mutex
	plugins map[uuid.UUID]*Plugin
}

func newMockPluginStore() *mockPluginStore {
	return &mockPluginStore{plugins: make(map[uuid.UUID]*Plugin)}
}

func (m *mockPluginStore) SavePlugin(_ context.Context, p *Plugin) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.plugins[p.ID] = p
	return nil
}

func (m *mockPluginStore) DeletePlugin(_ context.Context, id uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.plugins[id]; !ok {
		return fmt.Errorf("plugin %s not found", id)
	}
	delete(m.plugins, id)
	return nil
}

func (m *mockPluginStore) ListPlugins(_ context.Context) ([]*Plugin, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]*Plugin, 0, len(m.plugins))
	for _, p := range m.plugins {
		out = append(out, p)
	}
	return out, nil
}

func TestPluginRegistry_WithStore_RegisterPersists(t *testing.T) {
	store := newMockPluginStore()
	r := NewPluginRegistry(store)

	p := &Plugin{
		Name:              "persisted-plugin",
		Endpoint:          "http://localhost:9000/rpc",
		SubscribedColumns: []string{"profile"},
	}
	if err := r.Register(p); err != nil {
		t.Fatalf("Register: %v", err)
	}

	// Verify plugin was persisted to the store
	stored, err := store.ListPlugins(context.Background())
	if err != nil {
		t.Fatalf("ListPlugins: %v", err)
	}
	if len(stored) != 1 {
		t.Fatalf("expected 1 stored plugin, got %d", len(stored))
	}
	if stored[0].Name != "persisted-plugin" {
		t.Errorf("stored name: got %q, want %q", stored[0].Name, "persisted-plugin")
	}
}

func TestPluginRegistry_WithStore_DeleteRemovesFromStore(t *testing.T) {
	store := newMockPluginStore()
	r := NewPluginRegistry(store)

	p := &Plugin{
		Name:              "to-delete",
		Endpoint:          "http://localhost:9000/rpc",
		SubscribedColumns: []string{"profile"},
	}
	if err := r.Register(p); err != nil {
		t.Fatalf("Register: %v", err)
	}

	if err := r.Delete(p.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	stored, err := store.ListPlugins(context.Background())
	if err != nil {
		t.Fatalf("ListPlugins: %v", err)
	}
	if len(stored) != 0 {
		t.Errorf("expected 0 stored plugins after delete, got %d", len(stored))
	}
}

func TestPluginRegistry_LoadAll(t *testing.T) {
	store := newMockPluginStore()

	// Pre-populate the store (simulating data from a previous run)
	existing := &Plugin{
		ID:                uuid.New(),
		Name:              "pre-existing",
		Endpoint:          "http://localhost:9000/rpc",
		SubscribedColumns: []string{"orders"},
		Status:            PluginStatusActive,
	}
	if err := store.SavePlugin(context.Background(), existing); err != nil {
		t.Fatalf("SavePlugin: %v", err)
	}

	// Create a new registry and load from store
	r := NewPluginRegistry(store)
	if err := r.LoadAll(context.Background()); err != nil {
		t.Fatalf("LoadAll: %v", err)
	}

	// Verify the registry has the pre-existing plugin
	got, err := r.Get(existing.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Name != "pre-existing" {
		t.Errorf("Name: got %q, want %q", got.Name, "pre-existing")
	}

	plugins := r.List()
	if len(plugins) != 1 {
		t.Errorf("List: got %d, want 1", len(plugins))
	}
}

func TestPluginRegistry_LoadAll_NoStore(t *testing.T) {
	r := NewPluginRegistry()
	// LoadAll should be a no-op without a store
	if err := r.LoadAll(context.Background()); err != nil {
		t.Fatalf("LoadAll without store should not error: %v", err)
	}
}

func TestPluginRegistry_WithNilStore(t *testing.T) {
	// Passing nil explicitly should work like no store
	r := NewPluginRegistry(nil)

	p := &Plugin{
		Name:              "in-memory-only",
		Endpoint:          "http://localhost:9000/rpc",
		SubscribedColumns: []string{"profile"},
	}
	if err := r.Register(p); err != nil {
		t.Fatalf("Register: %v", err)
	}

	got, err := r.Get(p.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Name != "in-memory-only" {
		t.Errorf("Name: got %q", got.Name)
	}
}
