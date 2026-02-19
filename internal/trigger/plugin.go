package trigger

import (
	"context"
	"fmt"
	"slices"
	"sync"
	"time"

	"github.com/google/uuid"
)

// PluginStatus represents the activation state of a plugin.
type PluginStatus string

const (
	PluginStatusActive   PluginStatus = "active"
	PluginStatusInactive PluginStatus = "inactive"
)

// Plugin is an external JSON-RPC service that receives cell-write notifications.
type Plugin struct {
	ID                uuid.UUID    `json:"id"`
	Name              string       `json:"name"`
	Endpoint          string       `json:"endpoint"`
	SubscribedColumns []string     `json:"subscribed_columns"`
	Status            PluginStatus `json:"status"`
	CreatedAt         time.Time    `json:"created_at"`
}

// PluginRegistry is a thread-safe in-memory store of registered plugins.
// When a PluginStore is provided, mutations are persisted to durable storage.
type PluginRegistry struct {
	mu      sync.RWMutex
	plugins map[uuid.UUID]*Plugin
	store   PluginStore // optional; nil means in-memory only
}

// NewPluginRegistry creates an empty registry.
// An optional PluginStore enables write-through persistence.
func NewPluginRegistry(store ...PluginStore) *PluginRegistry {
	r := &PluginRegistry{plugins: make(map[uuid.UUID]*Plugin)}
	if len(store) > 0 && store[0] != nil {
		r.store = store[0]
	}
	return r
}

// LoadAll populates the in-memory registry from the backing store.
// It is a no-op if no store is configured.
func (r *PluginRegistry) LoadAll(ctx context.Context) error {
	if r.store == nil {
		return nil
	}
	plugins, err := r.store.ListPlugins(ctx)
	if err != nil {
		return fmt.Errorf("load plugins: %w", err)
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, p := range plugins {
		r.plugins[p.ID] = p
	}
	return nil
}

// Register adds a plugin to the registry. It assigns an ID and creation timestamp.
// It returns an error if a plugin with the same name is already registered.
func (r *PluginRegistry) Register(ctx context.Context, p *Plugin) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, existing := range r.plugins {
		if existing.Name == p.Name {
			return fmt.Errorf("plugin %q already registered", p.Name)
		}
	}
	p.ID = uuid.New()
	p.CreatedAt = time.Now()
	if p.Status == "" {
		p.Status = PluginStatusActive
	}
	if r.store != nil {
		if err := r.store.SavePlugin(ctx, p); err != nil {
			return fmt.Errorf("persist plugin: %w", err)
		}
	}
	r.plugins[p.ID] = p
	return nil
}

// Get returns a plugin by ID.
func (r *PluginRegistry) Get(id uuid.UUID) (*Plugin, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	p, ok := r.plugins[id]
	if !ok {
		return nil, fmt.Errorf("plugin %s not found", id)
	}
	return p, nil
}

// List returns all registered plugins.
func (r *PluginRegistry) List() []*Plugin {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]*Plugin, 0, len(r.plugins))
	for _, p := range r.plugins {
		out = append(out, p)
	}
	return out
}

// Delete removes a plugin by ID.
func (r *PluginRegistry) Delete(id uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.plugins[id]; !ok {
		return fmt.Errorf("plugin %s not found", id)
	}
	if r.store != nil {
		if err := r.store.DeletePlugin(context.Background(), id); err != nil {
			return fmt.Errorf("persist delete: %w", err)
		}
	}
	delete(r.plugins, id)
	return nil
}

// ForColumn returns all active plugins subscribed to the given column.
func (r *PluginRegistry) ForColumn(columnName string) []*Plugin {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var out []*Plugin
	for _, p := range r.plugins {
		if p.Status != PluginStatusActive {
			continue
		}
		if slices.Contains(p.SubscribedColumns, columnName) {
			out = append(out, p)
		}
	}
	return out
}

// Columns returns all unique column names across active plugins.
func (r *PluginRegistry) Columns() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	seen := make(map[string]struct{})
	for _, p := range r.plugins {
		if p.Status != PluginStatusActive {
			continue
		}
		for _, col := range p.SubscribedColumns {
			seen[col] = struct{}{}
		}
	}
	out := make([]string, 0, len(seen))
	for col := range seen {
		out = append(out, col)
	}
	return out
}
