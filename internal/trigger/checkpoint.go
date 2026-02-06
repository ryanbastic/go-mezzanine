package trigger

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/ryanbastic/go-mezzanine/internal/shard"
)

// PostgresCheckpoint implements Checkpoint using the trigger_checkpoints table.
// Each shard's checkpoint is stored on the backend that owns that shard.
type PostgresCheckpoint struct {
	pools map[shard.ID]*pgxpool.Pool
}

// NewPostgresCheckpoint creates a new PostgresCheckpoint with a per-shard pool mapping.
func NewPostgresCheckpoint(pools map[shard.ID]*pgxpool.Pool) *PostgresCheckpoint {
	return &PostgresCheckpoint{pools: pools}
}

func (c *PostgresCheckpoint) poolFor(shardID shard.ID) (*pgxpool.Pool, error) {
	pool, ok := c.pools[shardID]
	if !ok {
		return nil, fmt.Errorf("no pool for shard %d", shardID)
	}
	return pool, nil
}

func (c *PostgresCheckpoint) Load(ctx context.Context, shardID shard.ID, columnName string) (int64, error) {
	pool, err := c.poolFor(shardID)
	if err != nil {
		return 0, err
	}
	var addedID int64
	err = pool.QueryRow(ctx,
		`SELECT last_added_id FROM trigger_checkpoints WHERE shard_id = $1 AND column_name = $2`,
		int(shardID), columnName,
	).Scan(&addedID)
	if err != nil {
		// If no row exists, start from 0
		return 0, nil
	}
	return addedID, nil
}

func (c *PostgresCheckpoint) Save(ctx context.Context, shardID shard.ID, columnName string, addedID int64) error {
	pool, err := c.poolFor(shardID)
	if err != nil {
		return err
	}
	_, err = pool.Exec(ctx, `
		INSERT INTO trigger_checkpoints (shard_id, column_name, last_added_id, updated_at)
		VALUES ($1, $2, $3, now())
		ON CONFLICT (shard_id, column_name)
		DO UPDATE SET last_added_id = $3, updated_at = now()
	`, int(shardID), columnName, addedID)
	if err != nil {
		return fmt.Errorf("save checkpoint shard %d col %s: %w", shardID, columnName, err)
	}
	return nil
}
