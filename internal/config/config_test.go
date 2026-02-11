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
	for _, k := range []string{
		"PORT", "NUM_SHARDS", "LOG_LEVEL",
		"HTTP_READ_TIMEOUT", "HTTP_WRITE_TIMEOUT", "HTTP_IDLE_TIMEOUT",
		"DB_MAX_CONNS", "DB_MIN_CONNS", "DB_MAX_CONN_LIFETIME",
		"DB_MAX_CONN_IDLE_TIME", "DB_HEALTH_CHECK_PERIOD", "DB_QUERY_TIMEOUT",
	} {
		os.Unsetenv(k)
	}
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
	if cfg.IndexConfigPath != "" {
		t.Errorf("IndexConfigPath: got %q, want empty", cfg.IndexConfigPath)
	}

	// HTTP timeout defaults
	if cfg.HTTPReadTimeout != 5*time.Second {
		t.Errorf("HTTPReadTimeout: got %v, want %v", cfg.HTTPReadTimeout, 5*time.Second)
	}
	if cfg.HTTPWriteTimeout != 10*time.Second {
		t.Errorf("HTTPWriteTimeout: got %v, want %v", cfg.HTTPWriteTimeout, 10*time.Second)
	}
	if cfg.HTTPIdleTimeout != 120*time.Second {
		t.Errorf("HTTPIdleTimeout: got %v, want %v", cfg.HTTPIdleTimeout, 120*time.Second)
	}

	// DB pool defaults
	if cfg.DBMaxConns != 20 {
		t.Errorf("DBMaxConns: got %d, want %d", cfg.DBMaxConns, 20)
	}
	if cfg.DBMinConns != 2 {
		t.Errorf("DBMinConns: got %d, want %d", cfg.DBMinConns, 2)
	}
	if cfg.DBMaxConnLifetime != 30*time.Minute {
		t.Errorf("DBMaxConnLifetime: got %v, want %v", cfg.DBMaxConnLifetime, 30*time.Minute)
	}
	if cfg.DBMaxConnIdleTime != 5*time.Minute {
		t.Errorf("DBMaxConnIdleTime: got %v, want %v", cfg.DBMaxConnIdleTime, 5*time.Minute)
	}
	if cfg.DBHealthCheckPeriod != 30*time.Second {
		t.Errorf("DBHealthCheckPeriod: got %v, want %v", cfg.DBHealthCheckPeriod, 30*time.Second)
	}
	if cfg.DBQueryTimeout != 5*time.Second {
		t.Errorf("DBQueryTimeout: got %v, want %v", cfg.DBQueryTimeout, 5*time.Second)
	}
}

func TestLoad_CustomValues(t *testing.T) {
	envs := map[string]string{
		"SHARD_CONFIG_PATH":     "/custom/path.json",
		"INDEX_CONFIG_PATH":     "/custom/indexes.json",
		"PORT":                  "9090",
		"NUM_SHARDS":            "128",
		"LOG_LEVEL":             "debug",
		"HTTP_READ_TIMEOUT":     "15s",
		"HTTP_WRITE_TIMEOUT":    "30s",
		"HTTP_IDLE_TIMEOUT":     "60s",
		"DB_MAX_CONNS":          "50",
		"DB_MIN_CONNS":          "5",
		"DB_MAX_CONN_LIFETIME":  "1h",
		"DB_MAX_CONN_IDLE_TIME": "10m",
		"DB_HEALTH_CHECK_PERIOD": "1m",
		"DB_QUERY_TIMEOUT":       "3s",
	}
	for k, v := range envs {
		os.Setenv(k, v)
	}
	defer func() {
		for k := range envs {
			os.Unsetenv(k)
		}
	}()

	cfg := Load()

	if cfg.ShardConfigPath != "/custom/path.json" {
		t.Errorf("ShardConfigPath: got %q", cfg.ShardConfigPath)
	}
	if cfg.IndexConfigPath != "/custom/indexes.json" {
		t.Errorf("IndexConfigPath: got %q", cfg.IndexConfigPath)
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
	if cfg.HTTPReadTimeout != 15*time.Second {
		t.Errorf("HTTPReadTimeout: got %v", cfg.HTTPReadTimeout)
	}
	if cfg.HTTPWriteTimeout != 30*time.Second {
		t.Errorf("HTTPWriteTimeout: got %v", cfg.HTTPWriteTimeout)
	}
	if cfg.HTTPIdleTimeout != 60*time.Second {
		t.Errorf("HTTPIdleTimeout: got %v", cfg.HTTPIdleTimeout)
	}
	if cfg.DBMaxConns != 50 {
		t.Errorf("DBMaxConns: got %d", cfg.DBMaxConns)
	}
	if cfg.DBMinConns != 5 {
		t.Errorf("DBMinConns: got %d", cfg.DBMinConns)
	}
	if cfg.DBMaxConnLifetime != time.Hour {
		t.Errorf("DBMaxConnLifetime: got %v", cfg.DBMaxConnLifetime)
	}
	if cfg.DBMaxConnIdleTime != 10*time.Minute {
		t.Errorf("DBMaxConnIdleTime: got %v", cfg.DBMaxConnIdleTime)
	}
	if cfg.DBHealthCheckPeriod != time.Minute {
		t.Errorf("DBHealthCheckPeriod: got %v", cfg.DBHealthCheckPeriod)
	}
	if cfg.DBQueryTimeout != 3*time.Second {
		t.Errorf("DBQueryTimeout: got %v", cfg.DBQueryTimeout)
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
