package trigger

import (
	"context"

	"github.com/ryanbastic/go-mezzanine/internal/cell"
	"github.com/ryanbastic/go-mezzanine/internal/shard"
)

// HandlerFunc is invoked for each new cell in write order.
// Must be idempotent — may be called more than once for the same cell.
type HandlerFunc func(ctx context.Context, c cell.Cell) error

// Registration ties a column name to a handler.
type Registration struct {
	ColumnName string
	Handler    HandlerFunc
}

// Checkpoint persists/retrieves the last processed added_id per shard+column.
type Checkpoint interface {
	Load(ctx context.Context, shardID shard.ID, columnName string) (int64, error)
	Save(ctx context.Context, shardID shard.ID, columnName string, addedID int64) error
}
