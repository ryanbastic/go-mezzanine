package config

import (
	"os"
	"testing"
	"time"
)

func TestLoad_Defaults(t *testing.T) {
	// Set required env var
	os.Setenv("SHARD_CONFIG_PATH", "/tmp/shards.json")
	defer os.Unsetenv("SHARD_CONFIG_PATH")

	// Clear optional env vars to test defaults
	os.Unsetenv("PORT")
	os.Unsetenv("NUM_SHARDS")
	os.Unsetenv("LOG_LEVEL")
	os.Unsetenv("TRIGGER_POLL_INTERVAL")
	os.Unsetenv("TRIGGER_BATCH_SIZE")

	cfg := Load()

	if cfg.ShardConfigPath != "/tmp/shards.json" {
		t.Errorf("ShardConfigPath: got %q, want %q", cfg.ShardConfigPath, "/tmp/shards.json")
	}
	if cfg.Port != "8080" {
		t.Errorf("Port: got %q, want %q", cfg.Port, "8080")
	}
	if cfg.NumShards != 64 {
		t.Errorf("NumShards: got %d, want %d", cfg.NumShards, 64)
	}
	if cfg.LogLevel != "info" {
		t.Errorf("LogLevel: got %q, want %q", cfg.LogLevel, "info")
	}
	if cfg.TriggerPollInterval != 100*time.Millisecond {
		t.Errorf("TriggerPollInterval: got %v, want %v", cfg.TriggerPollInterval, 100*time.Millisecond)
	}
	if cfg.TriggerBatchSize != 100 {
		t.Errorf("TriggerBatchSize: got %d, want %d", cfg.TriggerBatchSize, 100)
	}
}

func TestLoad_CustomValues(t *testing.T) {
	os.Setenv("SHARD_CONFIG_PATH", "/custom/path.json")
	os.Setenv("PORT", "9090")
	os.Setenv("NUM_SHARDS", "128")
	os.Setenv("LOG_LEVEL", "debug")
	os.Setenv("TRIGGER_POLL_INTERVAL", "500ms")
	os.Setenv("TRIGGER_BATCH_SIZE", "50")
	defer func() {
		os.Unsetenv("SHARD_CONFIG_PATH")
		os.Unsetenv("PORT")
		os.Unsetenv("NUM_SHARDS")
		os.Unsetenv("LOG_LEVEL")
		os.Unsetenv("TRIGGER_POLL_INTERVAL")
		os.Unsetenv("TRIGGER_BATCH_SIZE")
	}()

	cfg := Load()

	if cfg.ShardConfigPath != "/custom/path.json" {
		t.Errorf("ShardConfigPath: got %q", cfg.ShardConfigPath)
	}
	if cfg.Port != "9090" {
		t.Errorf("Port: got %q", cfg.Port)
	}
	if cfg.NumShards != 128 {
		t.Errorf("NumShards: got %d", cfg.NumShards)
	}
	if cfg.LogLevel != "debug" {
		t.Errorf("LogLevel: got %q", cfg.LogLevel)
	}
	if cfg.TriggerPollInterval != 500*time.Millisecond {
		t.Errorf("TriggerPollInterval: got %v", cfg.TriggerPollInterval)
	}
	if cfg.TriggerBatchSize != 50 {
		t.Errorf("TriggerBatchSize: got %d", cfg.TriggerBatchSize)
	}
}

func TestLoad_MissingRequired_Panics(t *testing.T) {
	os.Unsetenv("SHARD_CONFIG_PATH")

	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for missing SHARD_CONFIG_PATH")
		}
	}()

	Load()
}

func TestGetEnv_Fallback(t *testing.T) {
	os.Unsetenv("TEST_NONEXISTENT_KEY")
	got := getEnv("TEST_NONEXISTENT_KEY", "default_value")
	if got != "default_value" {
		t.Errorf("got %q, want %q", got, "default_value")
	}
}

func TestGetEnv_Override(t *testing.T) {
	os.Setenv("TEST_GET_ENV_KEY", "override")
	defer os.Unsetenv("TEST_GET_ENV_KEY")

	got := getEnv("TEST_GET_ENV_KEY", "default")
	if got != "override" {
		t.Errorf("got %q, want %q", got, "override")
	}
}

func TestGetEnvInt_Fallback(t *testing.T) {
	os.Unsetenv("TEST_INT_NONEXISTENT")
	got := getEnvInt("TEST_INT_NONEXISTENT", 42)
	if got != 42 {
		t.Errorf("got %d, want %d", got, 42)
	}
}

func TestGetEnvInt_Valid(t *testing.T) {
	os.Setenv("TEST_INT_KEY", "99")
	defer os.Unsetenv("TEST_INT_KEY")

	got := getEnvInt("TEST_INT_KEY", 0)
	if got != 99 {
		t.Errorf("got %d, want %d", got, 99)
	}
}

func TestGetEnvInt_Invalid_ReturnsFallback(t *testing.T) {
	os.Setenv("TEST_INT_INVALID", "not_a_number")
	defer os.Unsetenv("TEST_INT_INVALID")

	got := getEnvInt("TEST_INT_INVALID", 7)
	if got != 7 {
		t.Errorf("got %d, want fallback %d", got, 7)
	}
}

func TestGetEnvDuration_Fallback(t *testing.T) {
	os.Unsetenv("TEST_DUR_NONEXISTENT")
	got := getEnvDuration("TEST_DUR_NONEXISTENT", 5*time.Second)
	if got != 5*time.Second {
		t.Errorf("got %v, want %v", got, 5*time.Second)
	}
}

func TestGetEnvDuration_Valid(t *testing.T) {
	os.Setenv("TEST_DUR_KEY", "2s")
	defer os.Unsetenv("TEST_DUR_KEY")

	got := getEnvDuration("TEST_DUR_KEY", 0)
	if got != 2*time.Second {
		t.Errorf("got %v, want %v", got, 2*time.Second)
	}
}

func TestGetEnvDuration_Invalid_ReturnsFallback(t *testing.T) {
	os.Setenv("TEST_DUR_INVALID", "not_a_duration")
	defer os.Unsetenv("TEST_DUR_INVALID")

	got := getEnvDuration("TEST_DUR_INVALID", 10*time.Millisecond)
	if got != 10*time.Millisecond {
		t.Errorf("got %v, want fallback %v", got, 10*time.Millisecond)
	}
}

func TestGetEnvRequired_Set(t *testing.T) {
	os.Setenv("TEST_REQUIRED_KEY", "hello")
	defer os.Unsetenv("TEST_REQUIRED_KEY")

	got := getEnvRequired("TEST_REQUIRED_KEY")
	if got != "hello" {
		t.Errorf("got %q, want %q", got, "hello")
	}
}

func TestGetEnvRequired_Empty_Panics(t *testing.T) {
	os.Unsetenv("TEST_REQUIRED_MISSING")

	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for missing required env var")
		}
	}()

	getEnvRequired("TEST_REQUIRED_MISSING")
}
