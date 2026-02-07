package config

import (
	"encoding/json"
	"fmt"
	"os"
)

// IndexDefinition describes a single secondary index to register at startup.
type IndexDefinition struct {
	Name          string   `json:"name"`
	SourceColumn  string   `json:"source_column"`
	ShardKeyField string   `json:"shard_key_field"`
	Fields        []string `json:"fields"`
}

// IndexConfig holds the list of secondary index definitions.
type IndexConfig struct {
	Indexes []IndexDefinition `json:"indexes"`
}

// LoadIndexConfig reads a JSON index config file and validates it.
func LoadIndexConfig(path string) (*IndexConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read index config: %w", err)
	}

	var cfg IndexConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse index config: %w", err)
	}

	if len(cfg.Indexes) == 0 {
		return nil, fmt.Errorf("index config: no indexes defined")
	}

	seen := make(map[string]bool, len(cfg.Indexes))
	for i, idx := range cfg.Indexes {
		if idx.Name == "" {
			return nil, fmt.Errorf("index config: index #%d has empty name", i)
		}
		if seen[idx.Name] {
			return nil, fmt.Errorf("index config: duplicate index name %q", idx.Name)
		}
		seen[idx.Name] = true
		if idx.SourceColumn == "" {
			return nil, fmt.Errorf("index config: index %q has empty source_column", idx.Name)
		}
		if idx.ShardKeyField == "" {
			return nil, fmt.Errorf("index config: index %q has empty shard_key_field", idx.Name)
		}
	}

	return &cfg, nil
}
