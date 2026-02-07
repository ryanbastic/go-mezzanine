package trigger

import (
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
type PluginRegistry struct {
	mu      sync.RWMutex
	plugins map[uuid.UUID]*Plugin
}

// NewPluginRegistry creates an empty registry.
func NewPluginRegistry() *PluginRegistry {
	return &PluginRegistry{plugins: make(map[uuid.UUID]*Plugin)}
}

// Register adds a plugin to the registry. It assigns an ID and creation timestamp.
func (r *PluginRegistry) Register(p *Plugin) {
	r.mu.Lock()
	defer r.mu.Unlock()
	p.ID = uuid.New()
	p.CreatedAt = time.Now()
	if p.Status == "" {
		p.Status = PluginStatusActive
	}
	r.plugins[p.ID] = p
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
