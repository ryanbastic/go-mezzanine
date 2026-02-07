package storage

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/ryanbastic/go-mezzanine/internal/cell"
)

// ErrCellNotFound is returned when a cell lookup finds no matching row.
var ErrCellNotFound = errors.New("cell not found")

// CellStore is the primary storage interface for a single shard.
type CellStore interface {
	// WriteCell inserts a new immutable cell. Returns the stored cell with added_id.
	WriteCell(ctx context.Context, req cell.WriteCellRequest) (*cell.Cell, error)

	// GetCell returns the cell at an exact (row_key, column_name, ref_key).
	GetCell(ctx context.Context, ref cell.CellRef) (*cell.Cell, error)

	// GetCellLatest returns the cell with the highest ref_key for (row_key, column_name).
	GetCellLatest(ctx context.Context, rowKey uuid.UUID, columnName string) (*cell.Cell, error)

	// GetRow returns the latest cell for every column_name in a row.
	GetRow(ctx context.Context, rowKey uuid.UUID) ([]cell.Cell, error)

	// PartitionRead reads a partition of cells.
	PartitionRead(ctx context.Context, partitionNumber int, readType int, addedID int64, createdAfter time.Time, limit int) ([]cell.Cell, error)

	// ScanCells returns cells with added_id > afterAddedID for a given column,
	// ordered by added_id ASC. Used by the trigger framework.
	ScanCells(ctx context.Context, columnName string, afterAddedID int64, limit int) ([]cell.Cell, error)
}
