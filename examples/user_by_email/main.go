package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/google/uuid"
	mezzanine "github.com/ryanbastic/go-mezzanine/pkg/mezzanine"
)

func main() {
	baseURL := "http://localhost:8080"
	if u := os.Getenv("MEZZANINE_URL"); u != "" {
		baseURL = u
	}

	cfg := mezzanine.NewConfiguration()
	cfg.Servers = mezzanine.ServerConfigurations{{URL: baseURL}}
	client := mezzanine.NewAPIClient(cfg)
	ctx := context.Background()

	// --- Step 1: Write a profile cell for Ryan Bastic ---
	//
	// The index config (indexes.json.example) defines:
	//   name:            user_by_email
	//   source_column:   profile
	//   shard_key_field: email       (determines index shard and lookup key)
	//   fields:          [email, display_name]
	//   unique_fields:   [email]     (enforces one row per email per shard)
	//
	// When a cell is written with column_name "profile", Mezzanine
	// automatically indexes it into the user_by_email index table.

	email := "ryan@bastic.net"
	rowKey := uuid.New().String()

	profileBody := map[string]any{
		"email":        email,
		"display_name": "Ryan Bastic",
	}

	cell := mezzanine.NewWriteCellBody(profileBody, "profile", 1, rowKey)
	writeResp, _, err := client.CellsAPI.WriteCell(ctx).WriteCellBody(*cell).Execute()
	if err != nil {
		log.Fatalf("failed to write profile cell: %v", err)
	}

	fmt.Println("=== Write Profile Cell ===")
	fmt.Printf("  row_key:     %s\n", writeResp.GetRowKey())
	fmt.Printf("  column_name: %s\n", writeResp.GetColumnName())
	fmt.Printf("  added_id:    %d\n", writeResp.GetAddedId())
	fmt.Printf("  body:        %v\n", writeResp.GetBody())
	fmt.Println()

	// --- Step 2: Read back the full row to confirm it was stored ---

	row, _, err := client.CellsAPI.GetRow(ctx, rowKey).Execute()
	if err != nil {
		log.Fatalf("failed to get row: %v", err)
	}

	fmt.Println("=== Get Row ===")
	fmt.Printf("  row_key: %s\n", row.GetRowKey())
	for _, c := range row.GetCells() {
		fmt.Printf("  cell: column=%s ref_key=%d body=%v\n",
			c.GetColumnName(), c.GetRefKey(), c.GetBody())
	}
	fmt.Println()

	// --- Step 3: Query the user_by_email index by email ---
	//
	// GET /v1/index/user_by_email/{email}
	//
	// This returns the index entry for the user with that email.
	// Each entry contains the denormalized email and display_name,
	// plus the row_key pointing back to the original cell.

	entries, _, err := client.IndexAPI.QueryIndex(ctx, "user_by_email", email).Execute()
	if err != nil {
		log.Fatalf("failed to query user_by_email index: %v", err)
	}

	fmt.Println("=== Query user_by_email Index ===")
	fmt.Printf("  email: %s\n", email)
	fmt.Printf("  entries found:      %d\n", len(entries))
	for _, e := range entries {
		body, _ := json.Marshal(e.GetBody())
		fmt.Printf("  [entry] added_id=%d row_key=%s body=%s\n",
			e.GetAddedId(), e.GetRowKey(), string(body))
	}
	if len(entries) == 0 {
		fmt.Println("  (no entries — is INDEX_CONFIG_PATH set?)")
	}
	fmt.Println()

	fmt.Println("Done.")
}
