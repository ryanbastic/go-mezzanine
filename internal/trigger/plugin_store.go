package trigger

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PluginStore is a persistent storage interface for trigger plugins.
type PluginStore interface {
	SavePlugin(ctx context.Context, p *Plugin) error
	DeletePlugin(ctx context.Context, id uuid.UUID) error
	ListPlugins(ctx context.Context) ([]*Plugin, error)
}

// PostgresPluginStore implements PluginStore backed by a PostgreSQL table.
type PostgresPluginStore struct {
	pool         *pgxpool.Pool
	queryTimeout time.Duration
}

// NewPostgresPluginStore creates a PluginStore using the given connection pool.
// queryTimeout sets the per-query context deadline; zero means no timeout.
func NewPostgresPluginStore(pool *pgxpool.Pool, queryTimeout time.Duration) *PostgresPluginStore {
	return &PostgresPluginStore{pool: pool, queryTimeout: queryTimeout}
}

func (s *PostgresPluginStore) withTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	if s.queryTimeout > 0 {
		return context.WithTimeout(ctx, s.queryTimeout)
	}
	return ctx, func() {}
}

func (s *PostgresPluginStore) SavePlugin(ctx context.Context, p *Plugin) error {
	ctx, cancel := s.withTimeout(ctx)
	defer cancel()

	_, err := s.pool.Exec(ctx, `
		INSERT INTO plugins (id, name, endpoint, subscribed_columns, status, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, p.ID, p.Name, p.Endpoint, p.SubscribedColumns, string(p.Status), p.CreatedAt)
	if err != nil {
		return fmt.Errorf("save plugin: %w", err)
	}
	return nil
}

func (s *PostgresPluginStore) DeletePlugin(ctx context.Context, id uuid.UUID) error {
	ctx, cancel := s.withTimeout(ctx)
	defer cancel()

	tag, err := s.pool.Exec(ctx, `DELETE FROM plugins WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete plugin: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("plugin %s not found", id)
	}
	return nil
}

func (s *PostgresPluginStore) ListPlugins(ctx context.Context) ([]*Plugin, error) {
	ctx, cancel := s.withTimeout(ctx)
	defer cancel()

	rows, err := s.pool.Query(ctx, `
		SELECT id, name, endpoint, subscribed_columns, status, created_at
		FROM plugins
		ORDER BY created_at ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("list plugins: %w", err)
	}
	defer rows.Close()

	var plugins []*Plugin
	for rows.Next() {
		p, err := scanPlugin(rows)
		if err != nil {
			return nil, err
		}
		plugins = append(plugins, p)
	}
	return plugins, rows.Err()
}

func scanPlugin(row pgx.Row) (*Plugin, error) {
	var p Plugin
	var status string
	if err := row.Scan(&p.ID, &p.Name, &p.Endpoint, &p.SubscribedColumns, &status, &p.CreatedAt); err != nil {
		return nil, fmt.Errorf("scan plugin: %w", err)
	}
	p.Status = PluginStatus(status)
	return &p, nil
}
