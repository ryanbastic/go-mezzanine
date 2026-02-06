package trigger

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/ryanbastic/go-mezzanine/internal/shard"
)

// PostgresCheckpoint implements Checkpoint using the trigger_checkpoints table.
type PostgresCheckpoint struct {
	pool *pgxpool.Pool
}

// NewPostgresCheckpoint creates a new PostgresCheckpoint.
func NewPostgresCheckpoint(pool *pgxpool.Pool) *PostgresCheckpoint {
	return &PostgresCheckpoint{pool: pool}
}

func (c *PostgresCheckpoint) Load(ctx context.Context, shardID shard.ID, columnName string) (int64, error) {
	var addedID int64
	err := c.pool.QueryRow(ctx,
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
	_, err := c.pool.Exec(ctx, `
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
