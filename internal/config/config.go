package config

import (
	"log/slog"
	"os"
	"strconv"
	"time"
)

type Config struct {
	ShardConfigPath string
	IndexConfigPath string
	Port            string
	NumShards   int
	LogLevel    string

	// HTTP server timeouts
	HTTPReadTimeout  time.Duration
	HTTPWriteTimeout time.Duration
	HTTPIdleTimeout  time.Duration

	// Database connection pool
	DBMaxConns          int
	DBMinConns          int
	DBMaxConnLifetime   time.Duration
	DBMaxConnIdleTime   time.Duration
	DBHealthCheckPeriod time.Duration
	DBQueryTimeout      time.Duration

	// Trigger framework
	TriggerRetryMax     int
	TriggerRetryBackoff time.Duration
	TriggerRPCTimeout   time.Duration

}

func Load() Config {
	return Config{
		ShardConfigPath: getEnvRequired("SHARD_CONFIG_PATH"),
		IndexConfigPath: getEnv("INDEX_CONFIG_PATH", ""),
		Port:            getEnv("PORT", "8080"),
		NumShards:       getEnvInt("NUM_SHARDS", 64),
		LogLevel:        getEnv("LOG_LEVEL", "info"),

		HTTPReadTimeout:  getEnvDuration("HTTP_READ_TIMEOUT", 5*time.Second),
		HTTPWriteTimeout: getEnvDuration("HTTP_WRITE_TIMEOUT", 10*time.Second),
		HTTPIdleTimeout:  getEnvDuration("HTTP_IDLE_TIMEOUT", 120*time.Second),

		DBMaxConns:          getEnvInt("DB_MAX_CONNS", 20),
		DBMinConns:          getEnvInt("DB_MIN_CONNS", 2),
		DBMaxConnLifetime:   getEnvDuration("DB_MAX_CONN_LIFETIME", 30*time.Minute),
		DBMaxConnIdleTime:   getEnvDuration("DB_MAX_CONN_IDLE_TIME", 5*time.Minute),
		DBHealthCheckPeriod: getEnvDuration("DB_HEALTH_CHECK_PERIOD", 30*time.Second),
		DBQueryTimeout:      getEnvDuration("DB_QUERY_TIMEOUT", 5*time.Second),

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
