package main

import (
	"context"
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

	users := []struct {
		Name  string
		Email string
		Role  string
	}{
		{Name: "Alice Johnson", Email: "alice@example.com", Role: "admin"},
		{Name: "Bob Smith", Email: "bob@example.com", Role: "editor"},
		{Name: "Carol White", Email: "carol@example.com", Role: "viewer"},
		{Name: "Dave Brown", Email: "dave@example.com", Role: "editor"},
		{Name: "Eve Davis", Email: "eve@example.com", Role: "admin"},
	}

	fmt.Printf("Seeding %d users to %s...\n", len(users), baseURL)

	for _, u := range users {
		rowKey := uuid.New().String()

		// Write a "profile" cell for each user
		profileBody := map[string]any{
			"name":  u.Name,
			"email": u.Email,
			"role":  u.Role,
		}
		cell := mezzanine.NewWriteCellBody(profileBody, "profile", 1, rowKey)
		resp, _, err := client.CellsAPI.WriteCell(ctx).WriteCellBody(*cell).Execute()
		if err != nil {
			log.Fatalf("failed to write profile for %s: %v", u.Name, err)
		}
		fmt.Printf("  [profile] %s  row_key=%s  added_id=%d\n", u.Name, rowKey, resp.GetAddedId())

		// Write a "settings" cell for each user
		settingsBody := map[string]any{
			"theme":         "dark",
			"notifications": true,
			"language":      "en",
		}
		settingsCell := mezzanine.NewWriteCellBody(settingsBody, "settings", 1, rowKey)
		resp, _, err = client.CellsAPI.WriteCell(ctx).WriteCellBody(*settingsCell).Execute()
		if err != nil {
			log.Fatalf("failed to write settings for %s: %v", u.Name, err)
		}
		fmt.Printf("  [settings] %s  added_id=%d\n", u.Name, resp.GetAddedId())

		// Read back the full row
		row, _, err := client.CellsAPI.GetRow(ctx, rowKey).Execute()
		if err != nil {
			log.Fatalf("failed to get row for %s: %v", u.Name, err)
		}
		fmt.Printf("  [row] %s  columns=%d\n\n", u.Name, len(row.GetCells()))
	}

	fmt.Println("Done. Seeded all users.")
}
