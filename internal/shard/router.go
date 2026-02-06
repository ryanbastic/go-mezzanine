package shard

import (
	"fmt"
	"sync"

	"github.com/ryanbastic/go-mezzanine/internal/storage"
)

// Router maps shard IDs to CellStore instances.
type Router struct {
	mu     sync.RWMutex
	stores map[ID]storage.CellStore
}

func NewRouter() *Router {
	return &Router{stores: make(map[ID]storage.CellStore)}
}

// Register associates a shard ID with a CellStore.
func (r *Router) Register(id ID, store storage.CellStore) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.stores[id] = store
}

// StoreFor returns the CellStore for the given shard ID.
func (r *Router) StoreFor(id ID) (storage.CellStore, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	s, ok := r.stores[id]
	if !ok {
		return nil, fmt.Errorf("no store registered for shard %d", id)
	}
	return s, nil
}
