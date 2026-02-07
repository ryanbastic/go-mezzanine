package config

import (
	"log/slog"
	"os"
	"strconv"
	"time"
)

type Config struct {
	ShardConfigPath string
	Port            string
	NumShards   int
	LogLevel    string

	// Trigger framework
	TriggerRetryMax     int
	TriggerRetryBackoff time.Duration
	TriggerRPCTimeout   time.Duration

}

func Load() Config {
	return Config{
		ShardConfigPath:     getEnvRequired("SHARD_CONFIG_PATH"),
		Port:                getEnv("PORT", "8080"),
		NumShards:           getEnvInt("NUM_SHARDS", 64),
		LogLevel:            getEnv("LOG_LEVEL", "info"),
		TriggerRetryMax:     getEnvInt("TRIGGER_RETRY_MAX", 3),
		TriggerRetryBackoff: getEnvDuration("TRIGGER_RETRY_BACKOFF", 100*time.Millisecond),
		TriggerRPCTimeout:   getEnvDuration("TRIGGER_RPC_TIMEOUT", 5*time.Second),
	}
}

func getEnvRequired(key string) string {
	v := os.Getenv(key)
	if v == "" {
		panic("required environment variable " + key + " is not set")
	}
	return v
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil {
			slog.Warn("invalid integer env var, using default", "key", key, "value", v, "error", err)
			return fallback
		}
		return n
	}
	return fallback
}

func getEnvDuration(key string, fallback time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		d, err := time.ParseDuration(v)
		if err != nil {
			slog.Warn("invalid duration env var, using default", "key", key, "value", v, "error", err)
			return fallback
		}
		return d
	}
	return fallback
}
