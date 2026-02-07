package index

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/ryanbastic/go-mezzanine/internal/cell"
	"github.com/ryanbastic/go-mezzanine/internal/shard"
)

// Entry is a single row in a secondary index table.
type Entry struct {
	AddedID   int64           `json:"added_id"`
	ShardKey  uuid.UUID       `json:"shard_key"`
	RowKey    uuid.UUID       `json:"row_key"`
	Body      json.RawMessage `json:"body"`
	CreatedAt time.Time       `json:"created_at"`
}

// Definition describes a secondary index.
type Definition struct {
	Name          string   // index table name (e.g., "user_by_email")
	SourceColumn  string   // column_name on the entity that triggers index updates
	ShardKeyField string   // JSON field path in the body used for sharding the index
	Fields        []string // JSON fields to denormalize into index body
}

// Store handles secondary index operations for a single shard.
type Store struct {
	pool  *pgxpool.Pool
	table string
}

// NewStore creates an index Store for a specific shard.
func NewStore(pool *pgxpool.Pool, indexName string, shardID int) *Store {
	return &Store{
		pool:  pool,
		table: IndexTable(indexName, shardID),
	}
}

// IndexTable returns the table name for a given index and shard.
func IndexTable(indexName string, shardID int) string {
	return fmt.Sprintf("index_%s_%04d", indexName, shardID)
}

// WriteEntry inserts a denormalized entry into the index.
func (s *Store) WriteEntry(ctx context.Context, entry Entry) error {
	query := fmt.Sprintf(`
		INSERT INTO %s (shard_key, row_key, body)
		VALUES ($1, $2, $3)
	`, s.table)

	_, err := s.pool.Exec(ctx, query, entry.ShardKey, entry.RowKey, entry.Body)
	if err != nil {
		return fmt.Errorf("write index entry: %w", err)
	}
	return nil
}

// QueryByShardKey returns all index entries for a given shard key.
func (s *Store) QueryByShardKey(ctx context.Context, shardKey uuid.UUID) ([]Entry, error) {
	query := fmt.Sprintf(`
		SELECT added_id, shard_key, row_key, body, created_at
		FROM %s
		WHERE shard_key = $1
		ORDER BY added_id ASC
	`, s.table)

	rows, err := s.pool.Query(ctx, query, shardKey)
	if err != nil {
		return nil, fmt.Errorf("query index: %w", err)
	}
	defer rows.Close()

	var entries []Entry
	for rows.Next() {
		var e Entry
		if err := rows.Scan(&e.AddedID, &e.ShardKey, &e.RowKey, &e.Body, &e.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan index entry: %w", err)
		}
		entries = append(entries, e)
	}
	return entries, rows.Err()
}

// Registry holds all index definitions and their per-shard stores.
type Registry struct {
	definitions map[string]Definition
	stores      map[string]map[shard.ID]*Store // indexName -> shardID -> Store
}

// NewRegistry creates an empty index Registry.
func NewRegistry() *Registry {
	return &Registry{
		definitions: make(map[string]Definition),
		stores:      make(map[string]map[shard.ID]*Store),
	}
}

// Register adds an index definition and creates stores for all shards.
func (r *Registry) Register(pool *pgxpool.Pool, def Definition, numShards int) {
	r.definitions[def.Name] = def
	shardStores := make(map[shard.ID]*Store, numShards)
	for i := range numShards {
		shardStores[shard.ID(i)] = NewStore(pool, def.Name, i)
	}
	r.stores[def.Name] = shardStores
}

// StoreFor returns the index store for a given index name and shard ID.
func (r *Registry) StoreFor(indexName string, shardID shard.ID) (*Store, bool) {
	shardStores, ok := r.stores[indexName]
	if !ok {
		return nil, false
	}
	store, ok := shardStores[shardID]
	return store, ok
}

// Definition returns the definition for a given index name.
func (r *Registry) GetDefinition(indexName string) (Definition, bool) {
	def, ok := r.definitions[indexName]
	return def, ok
}

// ForColumn returns all definitions whose SourceColumn matches columnName.
func (r *Registry) ForColumn(columnName string) []Definition {
	var defs []Definition
	for _, def := range r.definitions {
		if def.SourceColumn == columnName {
			defs = append(defs, def)
		}
	}
	return defs
}

// IndexCell finds matching index definitions for the cell's column and writes
// denormalized entries into the appropriate index shards.
func (r *Registry) IndexCell(ctx context.Context, c *cell.Cell, numShards int) error {
	defs := r.ForColumn(c.ColumnName)
	for _, def := range defs {
		shardKeyUUID, err := extractUUID(c.Body, def.ShardKeyField)
		if err != nil {
			return fmt.Errorf("index %s: extract shard key: %w", def.Name, err)
		}

		body, err := extractFields(c.Body, def.Fields)
		if err != nil {
			return fmt.Errorf("index %s: extract fields: %w", def.Name, err)
		}

		shardID := shard.ForRowKey(shardKeyUUID, numShards)
		store, ok := r.StoreFor(def.Name, shardID)
		if !ok {
			return fmt.Errorf("index %s: no store for shard %d", def.Name, shardID)
		}

		if err := store.WriteEntry(ctx, Entry{
			ShardKey: shardKeyUUID,
			RowKey:   c.RowKey,
			Body:     body,
		}); err != nil {
			return fmt.Errorf("index %s: %w", def.Name, err)
		}
	}
	return nil
}

// extractUUID reads a string field from a JSON object and parses it as a UUID.
func extractUUID(body json.RawMessage, field string) (uuid.UUID, error) {
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(body, &obj); err != nil {
		return uuid.Nil, fmt.Errorf("unmarshal body: %w", err)
	}

	raw, ok := obj[field]
	if !ok {
		return uuid.Nil, fmt.Errorf("field %q not found", field)
	}

	var s string
	if err := json.Unmarshal(raw, &s); err != nil {
		return uuid.Nil, fmt.Errorf("field %q is not a string: %w", field, err)
	}

	id, err := uuid.Parse(s)
	if err != nil {
		return uuid.Nil, fmt.Errorf("field %q is not a valid UUID: %w", field, err)
	}
	return id, nil
}

// extractFields copies only the specified keys from a JSON object.
func extractFields(body json.RawMessage, fields []string) (json.RawMessage, error) {
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(body, &obj); err != nil {
		return nil, fmt.Errorf("unmarshal body: %w", err)
	}

	subset := make(map[string]json.RawMessage, len(fields))
	for _, f := range fields {
		if v, ok := obj[f]; ok {
			subset[f] = v
		}
	}

	return json.Marshal(subset)
}

// CreateTables creates the index tables for all registered indexes.
func (r *Registry) CreateTables(ctx context.Context, pool *pgxpool.Pool, numShards int) error {
	for indexName := range r.definitions {
		for i := range numShards {
			table := IndexTable(indexName, i)
			ddl := fmt.Sprintf(`
				CREATE TABLE IF NOT EXISTS %s (
					added_id   BIGSERIAL PRIMARY KEY,
					shard_key  UUID NOT NULL,
					row_key    UUID NOT NULL,
					body       JSONB NOT NULL,
					created_at TIMESTAMPTZ NOT NULL DEFAULT now()
				);

				CREATE INDEX IF NOT EXISTS idx_%s_shard_key
					ON %s (shard_key);
			`, table, table, table)

			if _, err := pool.Exec(ctx, ddl); err != nil {
				return fmt.Errorf("create index table %s: %w", table, err)
			}
		}
	}
	return nil
}
