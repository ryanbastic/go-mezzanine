package storage

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/ryanbastic/go-mezzanine/internal/cell"
)

// ErrCellNotFound is returned when a cell lookup finds no matching row.
var ErrCellNotFound = errors.New("cell not found")

// Page represents a paginated result set with a cursor for the next page.
type Page struct {
	Cells      []cell.Cell
	NextCursor string // Empty if no more pages
	HasMore    bool
}

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

	// PartitionRead reads a partition of cells with optional pagination.
	// If limit > 0, at most limit cells are returned.
	// If cursor is non-empty, reading starts after the cursor position.
	// Returns a Page containing cells and a cursor for the next page (if more exist).
	PartitionRead(ctx context.Context, partitionNumber int, readType int, cursor string, limit int) (*Page, error)

	// ScanCells returns cells with added_id > afterAddedID for a given column,
	// ordered by added_id ASC. Used by the trigger framework.
	ScanCells(ctx context.Context, columnName string, afterAddedID int64, limit int) ([]cell.Cell, error)
}
