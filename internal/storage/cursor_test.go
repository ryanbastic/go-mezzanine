package storage

import (
	"testing"
)

func TestCursor_EncodeDecode(t *testing.T) {
	tests := []struct {
		name string
		c    Cursor
	}{
		{
			name: "added_id only",
			c:    Cursor{AddedID: 12345},
		},
		{
			name: "created_at only",
			c:    Cursor{CreatedAt: "2026-02-15T10:30:00Z"},
		},
		{
			name: "both fields",
			c:    Cursor{AddedID: 99999, CreatedAt: "2026-02-15T10:30:00Z"},
		},
		{
			name: "zero values",
			c:    Cursor{},
		},
		{
			name: "large added_id",
			c:    Cursor{AddedID: 9223372036854775807},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encoded, err := tt.c.Encode()
			if err != nil {
				t.Fatalf("Encode failed: %v", err)
			}

			decoded, err := DecodeCursor(encoded)
			if err != nil {
				t.Fatalf("Decode failed: %v", err)
			}

			if decoded.AddedID != tt.c.AddedID {
				t.Errorf("AddedID: got %d, want %d", decoded.AddedID, tt.c.AddedID)
			}
			if decoded.CreatedAt != tt.c.CreatedAt {
				t.Errorf("CreatedAt: got %q, want %q", decoded.CreatedAt, tt.c.CreatedAt)
			}
		})
	}
}

func TestDecodeCursor_Invalid(t *testing.T) {
	tests := []struct {
		name string
		input string
	}{
		{
			name: "invalid base64",
			input: "!!!invalid!!!",
		},
		{
			name: "invalid json",
			input: "eyJhZGRlZF9pZCI6bnVsbH0=", // {"added_id":null}
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := DecodeCursor(tt.input)
			if err == nil {
				t.Error("expected error, got nil")
			}
		})
	}
}
