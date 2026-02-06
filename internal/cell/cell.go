package cell

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// CellRef uniquely identifies a cell in the 3D hash map.
type CellRef struct {
	RowKey     uuid.UUID `json:"row_key"`
	ColumnName string    `json:"column_name"`
	RefKey     int64     `json:"ref_key"`
}

// Cell is an immutable JSON blob stored at a CellRef coordinate.
type Cell struct {
	AddedID    int64           `json:"added_id"`
	RowKey     uuid.UUID       `json:"row_key"`
	ColumnName string          `json:"column_name"`
	RefKey     int64           `json:"ref_key"`
	Body       json.RawMessage `json:"body"`
	CreatedAt  time.Time       `json:"created_at"`
}

// WriteCellRequest is what the caller provides to write a new cell.
type WriteCellRequest struct {
	RowKey     uuid.UUID       `json:"row_key"`
	ColumnName string          `json:"column_name"`
	RefKey     int64           `json:"ref_key"`
	Body       json.RawMessage `json:"body"`
}
