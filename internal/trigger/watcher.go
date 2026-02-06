package trigger

import (
	"context"
	"log/slog"
	"time"

	"github.com/ryanbastic/go-mezzanine/internal/shard"
	"github.com/ryanbastic/go-mezzanine/internal/storage"
)

// Watcher polls shards for new cells and fires registered trigger handlers.
type Watcher struct {
	registry     *Registry
	checkpoint   Checkpoint
	stores       map[shard.ID]storage.CellStore
	numShards    int
	pollInterval time.Duration
	batchSize    int
	logger       *slog.Logger
}

// NewWatcher creates a new trigger Watcher.
func NewWatcher(
	registry *Registry,
	checkpoint Checkpoint,
	stores map[shard.ID]storage.CellStore,
	numShards int,
	pollInterval time.Duration,
	batchSize int,
	logger *slog.Logger,
) *Watcher {
	return &Watcher{
		registry:     registry,
		checkpoint:   checkpoint,
		stores:       stores,
		numShards:    numShards,
		pollInterval: pollInterval,
		batchSize:    batchSize,
		logger:       logger,
	}
}

// Start launches goroutines to watch all shards for all registered columns.
// Returns when ctx is cancelled.
func (w *Watcher) Start(ctx context.Context) {
	columns := w.registry.Columns()
	if len(columns) == 0 {
		w.logger.Info("no trigger handlers registered, watcher idle")
		return
	}

	for shardNum := 0; shardNum < w.numShards; shardNum++ {
		for _, col := range columns {
			go w.watchShard(ctx, shard.ID(shardNum), col)
		}
	}
}

func (w *Watcher) watchShard(ctx context.Context, shardID shard.ID, columnName string) {
	store, ok := w.stores[shardID]
	if !ok {
		w.logger.Error("no store for shard", "shard", shardID)
		return
	}

	lastAddedID, err := w.checkpoint.Load(ctx, shardID, columnName)
	if err != nil {
		w.logger.Error("failed to load checkpoint", "shard", shardID, "column", columnName, "error", err)
		return
	}

	w.logger.Info("trigger watcher started", "shard", shardID, "column", columnName, "from_added_id", lastAddedID)

	ticker := time.NewTicker(w.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			// Persist final checkpoint before exit
			if err := w.checkpoint.Save(context.Background(), shardID, columnName, lastAddedID); err != nil {
				w.logger.Error("failed to save final checkpoint", "shard", shardID, "column", columnName, "error", err)
			}
			return
		case <-ticker.C:
			newLastID, err := w.processBatch(ctx, store, shardID, columnName, lastAddedID)
			if err != nil {
				w.logger.Error("trigger batch failed", "shard", shardID, "column", columnName, "error", err)
				continue
			}
			if newLastID > lastAddedID {
				lastAddedID = newLastID
				if err := w.checkpoint.Save(ctx, shardID, columnName, lastAddedID); err != nil {
					w.logger.Error("failed to save checkpoint", "shard", shardID, "column", columnName, "error", err)
				}
			}
		}
	}
}

func (w *Watcher) processBatch(
	ctx context.Context,
	store storage.CellStore,
	shardID shard.ID,
	columnName string,
	afterAddedID int64,
) (int64, error) {
	cells, err := store.ScanCells(ctx, columnName, afterAddedID, w.batchSize)
	if err != nil {
		return afterAddedID, err
	}

	handlers := w.registry.HandlersFor(columnName)
	lastID := afterAddedID

	for _, c := range cells {
		for _, handler := range handlers {
			if err := handler(ctx, c); err != nil {
				w.logger.Error("trigger handler failed",
					"shard", shardID,
					"column", columnName,
					"added_id", c.AddedID,
					"error", err,
				)
				// Continue processing — handlers must be idempotent and
				// failed cells will be retried on next poll since we don't
				// advance the checkpoint past them.
				// However, we stop this batch to maintain ordering.
				return lastID, nil
			}
		}
		lastID = c.AddedID
	}

	return lastID, nil
}
