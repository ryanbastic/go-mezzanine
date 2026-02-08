package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeTempIndexConfig(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "indexes.json")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("write temp index config: %v", err)
	}
	return path
}

func TestLoadIndexConfig_Valid_SingleIndex(t *testing.T) {
	cfg := `{
		"indexes": [{
			"name": "user_by_email",
			"source_column": "profile",
			"shard_key_field": "email",
			"fields": ["email", "display_name"]
		}]
	}`
	path := writeTempIndexConfig(t, cfg)

	ic, err := LoadIndexConfig(path)
	if err != nil {
		t.Fatalf("LoadIndexConfig: %v", err)
	}
	if len(ic.Indexes) != 1 {
		t.Errorf("got %d indexes, want 1", len(ic.Indexes))
	}
	if ic.Indexes[0].Name != "user_by_email" {
		t.Errorf("got name %q, want %q", ic.Indexes[0].Name, "user_by_email")
	}
	if ic.Indexes[0].SourceColumn != "profile" {
		t.Errorf("got source_column %q, want %q", ic.Indexes[0].SourceColumn, "profile")
	}
	if ic.Indexes[0].ShardKeyField != "email" {
		t.Errorf("got shard_key_field %q, want %q", ic.Indexes[0].ShardKeyField, "email")
	}
	if len(ic.Indexes[0].Fields) != 2 {
		t.Errorf("got %d fields, want 2", len(ic.Indexes[0].Fields))
	}
}

func TestLoadIndexConfig_Valid_MultipleIndexes(t *testing.T) {
	cfg := `{
		"indexes": [
			{
				"name": "user_by_email",
				"source_column": "profile",
				"shard_key_field": "email",
				"fields": ["email"]
			},
			{
				"name": "order_by_customer",
				"source_column": "orders",
				"shard_key_field": "customer_id",
				"fields": ["total", "status"]
			}
		]
	}`
	path := writeTempIndexConfig(t, cfg)

	ic, err := LoadIndexConfig(path)
	if err != nil {
		t.Fatalf("LoadIndexConfig: %v", err)
	}
	if len(ic.Indexes) != 2 {
		t.Errorf("got %d indexes, want 2", len(ic.Indexes))
	}
}

func TestLoadIndexConfig_FileNotFound(t *testing.T) {
	_, err := LoadIndexConfig("/nonexistent/path.json")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestLoadIndexConfig_InvalidJSON(t *testing.T) {
	path := writeTempIndexConfig(t, `{invalid`)

	_, err := LoadIndexConfig(path)
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
	if !strings.Contains(err.Error(), "parse index config") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestLoadIndexConfig_NoIndexes(t *testing.T) {
	path := writeTempIndexConfig(t, `{"indexes": []}`)

	_, err := LoadIndexConfig(path)
	if err == nil {
		t.Fatal("expected error for no indexes")
	}
	if !strings.Contains(err.Error(), "no indexes defined") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestLoadIndexConfig_EmptyName(t *testing.T) {
	cfg := `{
		"indexes": [{
			"name": "",
			"source_column": "profile",
			"shard_key_field": "email",
			"fields": ["email"]
		}]
	}`
	path := writeTempIndexConfig(t, cfg)

	_, err := LoadIndexConfig(path)
	if err == nil {
		t.Fatal("expected error for empty name")
	}
	if !strings.Contains(err.Error(), "empty name") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestLoadIndexConfig_DuplicateName(t *testing.T) {
	cfg := `{
		"indexes": [
			{
				"name": "dup",
				"source_column": "profile",
				"shard_key_field": "email",
				"fields": ["email"]
			},
			{
				"name": "dup",
				"source_column": "orders",
				"shard_key_field": "customer_id",
				"fields": ["total"]
			}
		]
	}`
	path := writeTempIndexConfig(t, cfg)

	_, err := LoadIndexConfig(path)
	if err == nil {
		t.Fatal("expected error for duplicate name")
	}
	if !strings.Contains(err.Error(), "duplicate index name") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestLoadIndexConfig_EmptySourceColumn(t *testing.T) {
	cfg := `{
		"indexes": [{
			"name": "test",
			"source_column": "",
			"shard_key_field": "email",
			"fields": ["email"]
		}]
	}`
	path := writeTempIndexConfig(t, cfg)

	_, err := LoadIndexConfig(path)
	if err == nil {
		t.Fatal("expected error for empty source_column")
	}
	if !strings.Contains(err.Error(), "empty source_column") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestLoadIndexConfig_EmptyShardKeyField(t *testing.T) {
	cfg := `{
		"indexes": [{
			"name": "test",
			"source_column": "profile",
			"shard_key_field": "",
			"fields": ["email"]
		}]
	}`
	path := writeTempIndexConfig(t, cfg)

	_, err := LoadIndexConfig(path)
	if err == nil {
		t.Fatal("expected error for empty shard_key_field")
	}
	if !strings.Contains(err.Error(), "empty shard_key_field") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestLoadIndexConfig_UniqueFields(t *testing.T) {
	cfg := `{
		"indexes": [{
			"name": "user_by_email",
			"source_column": "profile",
			"shard_key_field": "email",
			"fields": ["email", "display_name"],
			"unique_fields": ["email"]
		}]
	}`
	path := writeTempIndexConfig(t, cfg)

	ic, err := LoadIndexConfig(path)
	if err != nil {
		t.Fatalf("LoadIndexConfig: %v", err)
	}
	if len(ic.Indexes[0].UniqueFields) != 1 {
		t.Errorf("got %d unique_fields, want 1", len(ic.Indexes[0].UniqueFields))
	}
	if ic.Indexes[0].UniqueFields[0] != "email" {
		t.Errorf("got unique_field %q, want %q", ic.Indexes[0].UniqueFields[0], "email")
	}
}

func TestLoadIndexConfig_NoUniqueFields_Succeeds(t *testing.T) {
	cfg := `{
		"indexes": [{
			"name": "test",
			"source_column": "profile",
			"shard_key_field": "email",
			"fields": ["email"]
		}]
	}`
	path := writeTempIndexConfig(t, cfg)

	ic, err := LoadIndexConfig(path)
	if err != nil {
		t.Fatalf("LoadIndexConfig: %v", err)
	}
	if len(ic.Indexes[0].UniqueFields) != 0 {
		t.Errorf("got %d unique_fields, want 0", len(ic.Indexes[0].UniqueFields))
	}
}

func TestLoadIndexConfig_EmptyFields_Succeeds(t *testing.T) {
	cfg := `{
		"indexes": [{
			"name": "test",
			"source_column": "profile",
			"shard_key_field": "email",
			"fields": []
		}]
	}`
	path := writeTempIndexConfig(t, cfg)

	ic, err := LoadIndexConfig(path)
	if err != nil {
		t.Fatalf("LoadIndexConfig: %v", err)
	}
	if len(ic.Indexes[0].Fields) != 0 {
		t.Errorf("got %d fields, want 0", len(ic.Indexes[0].Fields))
	}
}
