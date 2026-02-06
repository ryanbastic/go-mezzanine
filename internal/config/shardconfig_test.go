package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeTempConfig(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "shards.json")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("write temp config: %v", err)
	}
	return path
}

func TestLoadShardConfig_Valid_SingleBackend(t *testing.T) {
	cfg := `{
		"backends": [{
			"name": "primary",
			"database_url": "postgres://localhost/db",
			"shard_start": 0,
			"shard_end": 3
		}]
	}`
	path := writeTempConfig(t, cfg)

	sc, err := LoadShardConfig(path, 4)
	if err != nil {
		t.Fatalf("LoadShardConfig: %v", err)
	}
	if len(sc.Backends) != 1 {
		t.Errorf("got %d backends, want 1", len(sc.Backends))
	}
	if sc.Backends[0].Name != "primary" {
		t.Errorf("got name %q, want %q", sc.Backends[0].Name, "primary")
	}
}

func TestLoadShardConfig_Valid_MultipleBackends(t *testing.T) {
	cfg := `{
		"backends": [
			{
				"name": "backend-a",
				"database_url": "postgres://a/db",
				"shard_start": 0,
				"shard_end": 1
			},
			{
				"name": "backend-b",
				"database_url": "postgres://b/db",
				"shard_start": 2,
				"shard_end": 3
			}
		]
	}`
	path := writeTempConfig(t, cfg)

	sc, err := LoadShardConfig(path, 4)
	if err != nil {
		t.Fatalf("LoadShardConfig: %v", err)
	}
	if len(sc.Backends) != 2 {
		t.Errorf("got %d backends, want 2", len(sc.Backends))
	}
}

func TestLoadShardConfig_FileNotFound(t *testing.T) {
	_, err := LoadShardConfig("/nonexistent/path.json", 4)
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestLoadShardConfig_InvalidJSON(t *testing.T) {
	path := writeTempConfig(t, `{invalid`)

	_, err := LoadShardConfig(path, 4)
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
	if !strings.Contains(err.Error(), "parse shard config") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestLoadShardConfig_NoBackends(t *testing.T) {
	path := writeTempConfig(t, `{"backends": []}`)

	_, err := LoadShardConfig(path, 4)
	if err == nil {
		t.Fatal("expected error for no backends")
	}
	if !strings.Contains(err.Error(), "no backends defined") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestLoadShardConfig_EmptyDatabaseURL(t *testing.T) {
	cfg := `{
		"backends": [{
			"name": "bad",
			"database_url": "",
			"shard_start": 0,
			"shard_end": 3
		}]
	}`
	path := writeTempConfig(t, cfg)

	_, err := LoadShardConfig(path, 4)
	if err == nil {
		t.Fatal("expected error for empty database_url")
	}
	if !strings.Contains(err.Error(), "empty database_url") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestLoadShardConfig_NegativeShardRange(t *testing.T) {
	cfg := `{
		"backends": [{
			"name": "bad",
			"database_url": "postgres://localhost/db",
			"shard_start": -1,
			"shard_end": 3
		}]
	}`
	path := writeTempConfig(t, cfg)

	_, err := LoadShardConfig(path, 4)
	if err == nil {
		t.Fatal("expected error for negative shard range")
	}
	if !strings.Contains(err.Error(), "negative shard range") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestLoadShardConfig_StartGreaterThanEnd(t *testing.T) {
	cfg := `{
		"backends": [{
			"name": "bad",
			"database_url": "postgres://localhost/db",
			"shard_start": 5,
			"shard_end": 2
		}]
	}`
	path := writeTempConfig(t, cfg)

	_, err := LoadShardConfig(path, 10)
	if err == nil {
		t.Fatal("expected error for shard_start > shard_end")
	}
	if !strings.Contains(err.Error(), "shard_start") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestLoadShardConfig_ShardEndExceedsNumShards(t *testing.T) {
	cfg := `{
		"backends": [{
			"name": "bad",
			"database_url": "postgres://localhost/db",
			"shard_start": 0,
			"shard_end": 10
		}]
	}`
	path := writeTempConfig(t, cfg)

	_, err := LoadShardConfig(path, 4)
	if err == nil {
		t.Fatal("expected error for shard_end >= num_shards")
	}
	if !strings.Contains(err.Error(), "shard_end") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestLoadShardConfig_OverlappingShards(t *testing.T) {
	cfg := `{
		"backends": [
			{
				"name": "a",
				"database_url": "postgres://a/db",
				"shard_start": 0,
				"shard_end": 2
			},
			{
				"name": "b",
				"database_url": "postgres://b/db",
				"shard_start": 2,
				"shard_end": 3
			}
		]
	}`
	path := writeTempConfig(t, cfg)

	_, err := LoadShardConfig(path, 4)
	if err == nil {
		t.Fatal("expected error for overlapping shards")
	}
	if !strings.Contains(err.Error(), "covered by multiple backends") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestLoadShardConfig_UncoveredShard(t *testing.T) {
	cfg := `{
		"backends": [{
			"name": "partial",
			"database_url": "postgres://localhost/db",
			"shard_start": 0,
			"shard_end": 1
		}]
	}`
	path := writeTempConfig(t, cfg)

	_, err := LoadShardConfig(path, 4)
	if err == nil {
		t.Fatal("expected error for uncovered shards")
	}
	if !strings.Contains(err.Error(), "not covered by any backend") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestLoadShardConfig_SingleShard(t *testing.T) {
	cfg := `{
		"backends": [{
			"name": "only",
			"database_url": "postgres://localhost/db",
			"shard_start": 0,
			"shard_end": 0
		}]
	}`
	path := writeTempConfig(t, cfg)

	sc, err := LoadShardConfig(path, 1)
	if err != nil {
		t.Fatalf("LoadShardConfig: %v", err)
	}
	if len(sc.Backends) != 1 {
		t.Errorf("got %d backends, want 1", len(sc.Backends))
	}
}

func TestLoadShardConfig_ManyBackends(t *testing.T) {
	cfg := `{
		"backends": [
			{"name": "b0", "database_url": "postgres://0/db", "shard_start": 0, "shard_end": 0},
			{"name": "b1", "database_url": "postgres://1/db", "shard_start": 1, "shard_end": 1},
			{"name": "b2", "database_url": "postgres://2/db", "shard_start": 2, "shard_end": 2},
			{"name": "b3", "database_url": "postgres://3/db", "shard_start": 3, "shard_end": 3}
		]
	}`
	path := writeTempConfig(t, cfg)

	sc, err := LoadShardConfig(path, 4)
	if err != nil {
		t.Fatalf("LoadShardConfig: %v", err)
	}
	if len(sc.Backends) != 4 {
		t.Errorf("got %d backends, want 4", len(sc.Backends))
	}
}

func TestBackendConfig_Fields(t *testing.T) {
	cfg := `{
		"backends": [{
			"name": "test-backend",
			"database_url": "postgres://user:pass@host:5432/db",
			"shard_start": 0,
			"shard_end": 7
		}]
	}`
	path := writeTempConfig(t, cfg)

	sc, err := LoadShardConfig(path, 8)
	if err != nil {
		t.Fatalf("LoadShardConfig: %v", err)
	}

	b := sc.Backends[0]
	if b.Name != "test-backend" {
		t.Errorf("Name: got %q", b.Name)
	}
	if b.DatabaseURL != "postgres://user:pass@host:5432/db" {
		t.Errorf("DatabaseURL: got %q", b.DatabaseURL)
	}
	if b.ShardStart != 0 {
		t.Errorf("ShardStart: got %d", b.ShardStart)
	}
	if b.ShardEnd != 7 {
		t.Errorf("ShardEnd: got %d", b.ShardEnd)
	}
}
