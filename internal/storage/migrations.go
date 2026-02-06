package storage

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// RunMigrationsForPool creates shard cell tables for the given range and the
// trigger checkpoint table on a single backend pool.
func RunMigrationsForPool(ctx context.Context, pool *pgxpool.Pool, shardStart, shardEnd int) error {
	for i := shardStart; i <= shardEnd; i++ {
		table := ShardTable(i)
		ddl := fmt.Sprintf(`
			CREATE TABLE IF NOT EXISTS %s (
				added_id    BIGSERIAL PRIMARY KEY,
				row_key     UUID NOT NULL,
				column_name TEXT NOT NULL,
				ref_key     BIGINT NOT NULL,
				body        JSONB NOT NULL,
				created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),

				CONSTRAINT uq_%s_ref UNIQUE (row_key, column_name, ref_key)
			);

			CREATE INDEX IF NOT EXISTS idx_%s_row_col
				ON %s (row_key, column_name, ref_key DESC);

			CREATE INDEX IF NOT EXISTS idx_%s_trigger
				ON %s (column_name, added_id);
		`, table, table, table, table, table, table)

		if _, err := pool.Exec(ctx, ddl); err != nil {
			return fmt.Errorf("migrate shard %d: %w", i, err)
		}
	}

	checkpointDDL := `
		CREATE TABLE IF NOT EXISTS trigger_checkpoints (
			shard_id      INT NOT NULL,
			column_name   TEXT NOT NULL,
			last_added_id BIGINT NOT NULL DEFAULT 0,
			updated_at    TIMESTAMPTZ NOT NULL DEFAULT now(),

			PRIMARY KEY (shard_id, column_name)
		);
	`
	if _, err := pool.Exec(ctx, checkpointDDL); err != nil {
		return fmt.Errorf("migrate trigger_checkpoints: %w", err)
	}

	return nil
}

// ShardTable returns the table name for a given shard number.
func ShardTable(shardID int) string {
	return fmt.Sprintf("cells_%04d", shardID)
}
