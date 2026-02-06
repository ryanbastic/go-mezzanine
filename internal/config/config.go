package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	DatabaseURL string
	Port        string
	NumShards   int
	LogLevel    string

	// Trigger framework
	TriggerPollInterval time.Duration
	TriggerBatchSize    int

	// Circuit breaker
	CBMaxFailures  int
	CBResetTimeout time.Duration
}

func Load() Config {
	return Config{
		DatabaseURL:         getEnv("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/mezzanine?sslmode=disable"),
		Port:                getEnv("PORT", "8080"),
		NumShards:           getEnvInt("NUM_SHARDS", 64),
		LogLevel:            getEnv("LOG_LEVEL", "info"),
		TriggerPollInterval: getEnvDuration("TRIGGER_POLL_INTERVAL", 100*time.Millisecond),
		TriggerBatchSize:    getEnvInt("TRIGGER_BATCH_SIZE", 100),
		CBMaxFailures:       getEnvInt("CB_MAX_FAILURES", 5),
		CBResetTimeout:      getEnvDuration("CB_RESET_TIMEOUT", 30*time.Second),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return fallback
}

func getEnvDuration(key string, fallback time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return fallback
}
