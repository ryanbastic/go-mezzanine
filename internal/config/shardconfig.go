package config

import (
	"encoding/json"
	"fmt"
	"os"
)

// BackendConfig describes a single PostgreSQL backend and its shard range.
type BackendConfig struct {
	Name        string `json:"name"`
	DatabaseURL string `json:"database_url"`
	ShardStart  int    `json:"shard_start"`
	ShardEnd    int    `json:"shard_end"`
}

// ShardConfig holds the list of backends that together cover all shards.
type ShardConfig struct {
	Backends []BackendConfig `json:"backends"`
}

// LoadShardConfig reads a JSON shard config file and validates it against numShards.
func LoadShardConfig(path string, numShards int) (*ShardConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read shard config: %w", err)
	}

	var cfg ShardConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse shard config: %w", err)
	}

	if len(cfg.Backends) == 0 {
		return nil, fmt.Errorf("shard config: no backends defined")
	}

	covered := make([]bool, numShards)

	for i, b := range cfg.Backends {
		if b.DatabaseURL == "" {
			return nil, fmt.Errorf("shard config: backend %q (#%d) has empty database_url", b.Name, i)
		}
		if b.ShardStart < 0 || b.ShardEnd < 0 {
			return nil, fmt.Errorf("shard config: backend %q has negative shard range", b.Name)
		}
		if b.ShardStart > b.ShardEnd {
			return nil, fmt.Errorf("shard config: backend %q has shard_start (%d) > shard_end (%d)", b.Name, b.ShardStart, b.ShardEnd)
		}
		if b.ShardEnd >= numShards {
			return nil, fmt.Errorf("shard config: backend %q shard_end (%d) >= num_shards (%d)", b.Name, b.ShardEnd, numShards)
		}
		for s := b.ShardStart; s <= b.ShardEnd; s++ {
			if covered[s] {
				return nil, fmt.Errorf("shard config: shard %d is covered by multiple backends", s)
			}
			covered[s] = true
		}
	}

	for s := 0; s < numShards; s++ {
		if !covered[s] {
			return nil, fmt.Errorf("shard config: shard %d is not covered by any backend", s)
		}
	}

	return &cfg, nil
}
