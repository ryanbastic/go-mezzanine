package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/google/uuid"
	"github.com/ryanbastic/go-mezzanine/internal/storage"
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

	seedUsers(ctx, client, baseURL, users)
	partitionRead(ctx, client)
}

func partitionRead(ctx context.Context, client *mezzanine.APIClient) {
	fmt.Printf("Testing partition read...\n")

	numShards, _, err := client.ShardsAPI.GetShardCount(ctx).Execute()
	if err != nil {
		log.Fatalf("failed to get shard count: %v", err)
	}
	fmt.Printf("Number of shards: %d\n", numShards.NumShards)

	readTypeAddedID := int64(storage.PartitionReadTypeAddedID)
	for i := 0; i < int(numShards.NumShards); i++ {
		fmt.Printf("  Reading partition %d...\n", i)

		partReadReq := client.CellsAPI.PartitionRead(ctx).
			AddedId(0).
			Limit(100).
			ReadType(readTypeAddedID).
			PartitionNumber(int64(i))

		cells, _, err := partReadReq.Execute()
		if err != nil {
			log.Fatalf("failed to read partition %d: %v", i, err)
		}

		for _, cell := range cells {
			fmt.Printf("    [cell] row_key=%s  column_name=%s  ref_key=%d body=%+v added_id=%d\n", cell.GetRowKey(), cell.GetColumnName(), cell.GetRefKey(), cell.GetBody(), cell.GetAddedId())
		}
	}
}

func seedUsers(ctx context.Context, client *mezzanine.APIClient, baseURL string, users []struct {
	Name  string
	Email string
	Role  string
}) {

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
		fmt.Printf("  [profile] %s (body: %s) row_key=%s  added_id=%d\n", u.Name, resp.GetBody(), rowKey, resp.GetAddedId())

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
		fmt.Printf("  [settings] %s (body: %s) added_id=%d\n", u.Name, resp.GetBody(), resp.GetAddedId())

		// Read back the full row
		row, _, err := client.CellsAPI.GetRow(ctx, rowKey).Execute()
		if err != nil {
			log.Fatalf("failed to get row for %s: %v", u.Name, err)
		}
		fmt.Printf("  [row] %s  columns=%d\n\n", u.Name, len(row.GetCells()))
	}

	fmt.Println("Done. Seeded all users.")

}
