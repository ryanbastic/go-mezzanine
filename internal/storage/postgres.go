package storage

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/ryanbastic/go-mezzanine/internal/cell"
)

// PostgresStore implements CellStore for a single shard using PostgreSQL.
type PostgresStore struct {
	pool  *pgxpool.Pool
	table string
}

// NewPostgresStore creates a CellStore backed by a specific shard table.
func NewPostgresStore(pool *pgxpool.Pool, shardID int) *PostgresStore {
	return &PostgresStore{
		pool:  pool,
		table: ShardTable(shardID),
	}
}

func (s *PostgresStore) WriteCell(ctx context.Context, req cell.WriteCellRequest) (*cell.Cell, error) {
	query := fmt.Sprintf(`
		INSERT INTO %s (row_key, column_name, ref_key, body)
		VALUES ($1, $2, $3, $4)
		RETURNING added_id, row_key, column_name, ref_key, body, created_at
	`, s.table)

	var c cell.Cell
	err := s.pool.QueryRow(ctx, query,
		req.RowKey, req.ColumnName, req.RefKey, req.Body,
	).Scan(&c.AddedID, &c.RowKey, &c.ColumnName, &c.RefKey, &c.Body, &c.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("write cell: %w", err)
	}
	return &c, nil
}

func (s *PostgresStore) GetCell(ctx context.Context, ref cell.CellRef) (*cell.Cell, error) {
	query := fmt.Sprintf(`
		SELECT added_id, row_key, column_name, ref_key, body, created_at
		FROM %s
		WHERE row_key = $1 AND column_name = $2 AND ref_key = $3
	`, s.table)

	var c cell.Cell
	err := s.pool.QueryRow(ctx, query, ref.RowKey, ref.ColumnName, ref.RefKey).
		Scan(&c.AddedID, &c.RowKey, &c.ColumnName, &c.RefKey, &c.Body, &c.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrCellNotFound
		}
		return nil, fmt.Errorf("get cell: %w", err)
	}
	return &c, nil
}

func (s *PostgresStore) GetCellLatest(ctx context.Context, rowKey uuid.UUID, columnName string) (*cell.Cell, error) {
	query := fmt.Sprintf(`
		SELECT added_id, row_key, column_name, ref_key, body, created_at
		FROM %s
		WHERE row_key = $1 AND column_name = $2
		ORDER BY ref_key DESC
		LIMIT 1
	`, s.table)

	var c cell.Cell
	err := s.pool.QueryRow(ctx, query, rowKey, columnName).
		Scan(&c.AddedID, &c.RowKey, &c.ColumnName, &c.RefKey, &c.Body, &c.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrCellNotFound
		}
		return nil, fmt.Errorf("get cell latest: %w", err)
	}
	return &c, nil
}

func (s *PostgresStore) GetRow(ctx context.Context, rowKey uuid.UUID) ([]cell.Cell, error) {
	query := fmt.Sprintf(`
		SELECT DISTINCT ON (column_name)
			added_id, row_key, column_name, ref_key, body, created_at
		FROM %s
		WHERE row_key = $1
		ORDER BY column_name, ref_key DESC
	`, s.table)

	rows, err := s.pool.Query(ctx, query, rowKey)
	if err != nil {
		return nil, fmt.Errorf("get row: %w", err)
	}
	defer rows.Close()

	var cells []cell.Cell
	for rows.Next() {
		var c cell.Cell
		if err := rows.Scan(&c.AddedID, &c.RowKey, &c.ColumnName, &c.RefKey, &c.Body, &c.CreatedAt); err != nil {
			return nil, fmt.Errorf("get row scan: %w", err)
		}
		cells = append(cells, c)
	}
	return cells, rows.Err()
}

func (s *PostgresStore) ScanCells(ctx context.Context, columnName string, afterAddedID int64, limit int) ([]cell.Cell, error) {
	query := fmt.Sprintf(`
		SELECT added_id, row_key, column_name, ref_key, body, created_at
		FROM %s
		WHERE column_name = $1 AND added_id > $2
		ORDER BY added_id ASC
		LIMIT $3
	`, s.table)

	rows, err := s.pool.Query(ctx, query, columnName, afterAddedID, limit)
	if err != nil {
		return nil, fmt.Errorf("scan cells: %w", err)
	}
	defer rows.Close()

	var cells []cell.Cell
	for rows.Next() {
		var c cell.Cell
		if err := rows.Scan(&c.AddedID, &c.RowKey, &c.ColumnName, &c.RefKey, &c.Body, &c.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan cells scan: %w", err)
		}
		cells = append(cells, c)
	}
	return cells, rows.Err()
}
