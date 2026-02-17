package storage

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
)

// Cursor is an opaque pagination token for cursor-based pagination.
type Cursor struct {
	// AddedID is the cursor position for added_id-based pagination.
	AddedID int64 `json:"added_id,omitempty"`
	// CreatedAt is the cursor position for created_at-based pagination.
	CreatedAt string `json:"created_at,omitempty"`
}

// Encode serializes the cursor to a base64-encoded string.
func (c *Cursor) Encode() (string, error) {
	data, err := json.Marshal(c)
	if err != nil {
		return "", fmt.Errorf("marshal cursor: %w", err)
	}
	return base64.URLEncoding.EncodeToString(data), nil
}

// DecodeCursor parses a base64-encoded cursor string.
func DecodeCursor(s string) (*Cursor, error) {
	data, err := base64.URLEncoding.DecodeString(s)
	if err != nil {
		return nil, fmt.Errorf("decode cursor: %w", err)
	}
	var c Cursor
	if err := json.Unmarshal(data, &c); err != nil {
		return nil, fmt.Errorf("unmarshal cursor: %w", err)
	}
	return &c, nil
}
