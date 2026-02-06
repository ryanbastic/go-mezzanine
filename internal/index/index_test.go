package index

import (
	"testing"

	"github.com/ryanbastic/go-mezzanine/internal/shard"
)

func TestIndexTable(t *testing.T) {
	tests := []struct {
		name    string
		shardID int
		want    string
	}{
		{"user_by_email", 0, "index_user_by_email_0000"},
		{"user_by_email", 42, "index_user_by_email_0042"},
		{"orders", 1, "index_orders_0001"},
		{"x", 9999, "index_x_9999"},
	}

	for _, tt := range tests {
		got := IndexTable(tt.name, tt.shardID)
		if got != tt.want {
			t.Errorf("IndexTable(%q, %d) = %q, want %q", tt.name, tt.shardID, got, tt.want)
		}
	}
}

func TestNewRegistry(t *testing.T) {
	r := NewRegistry()
	if r == nil {
		t.Fatal("NewRegistry returned nil")
	}
}

func TestRegistry_Register_And_StoreFor(t *testing.T) {
	r := NewRegistry()

	def := Definition{
		Name:          "user_by_email",
		SourceColumn:  "profile",
		ShardKeyField: "email_hash",
		Fields:        []string{"email", "name"},
	}

	// Register with nil pool (we won't actually query)
	r.Register(nil, def, 4)

	// Verify StoreFor works
	for i := 0; i < 4; i++ {
		store, ok := r.StoreFor("user_by_email", shard.ID(i))
		if !ok {
			t.Errorf("StoreFor shard %d: not found", i)
		}
		if store == nil {
			t.Errorf("StoreFor shard %d: nil store", i)
		}
	}
}

func TestRegistry_StoreFor_UnknownIndex(t *testing.T) {
	r := NewRegistry()

	_, ok := r.StoreFor("nonexistent", shard.ID(0))
	if ok {
		t.Error("expected not found for nonexistent index")
	}
}

func TestRegistry_StoreFor_UnknownShard(t *testing.T) {
	r := NewRegistry()
	def := Definition{Name: "test_idx"}
	r.Register(nil, def, 2)

	_, ok := r.StoreFor("test_idx", shard.ID(99))
	if ok {
		t.Error("expected not found for unknown shard")
	}
}

func TestRegistry_GetDefinition(t *testing.T) {
	r := NewRegistry()

	def := Definition{
		Name:          "user_by_email",
		SourceColumn:  "profile",
		ShardKeyField: "email_hash",
		Fields:        []string{"email", "name"},
	}
	r.Register(nil, def, 2)

	got, ok := r.GetDefinition("user_by_email")
	if !ok {
		t.Fatal("definition not found")
	}
	if got.Name != "user_by_email" {
		t.Errorf("Name: got %q", got.Name)
	}
	if got.SourceColumn != "profile" {
		t.Errorf("SourceColumn: got %q", got.SourceColumn)
	}
	if got.ShardKeyField != "email_hash" {
		t.Errorf("ShardKeyField: got %q", got.ShardKeyField)
	}
	if len(got.Fields) != 2 {
		t.Errorf("Fields: got %d", len(got.Fields))
	}
}

func TestRegistry_GetDefinition_NotFound(t *testing.T) {
	r := NewRegistry()
	_, ok := r.GetDefinition("nonexistent")
	if ok {
		t.Error("expected not found")
	}
}

func TestRegistry_MultipleIndexes(t *testing.T) {
	r := NewRegistry()

	r.Register(nil, Definition{Name: "idx_a"}, 2)
	r.Register(nil, Definition{Name: "idx_b"}, 2)

	if _, ok := r.StoreFor("idx_a", shard.ID(0)); !ok {
		t.Error("idx_a shard 0 not found")
	}
	if _, ok := r.StoreFor("idx_b", shard.ID(1)); !ok {
		t.Error("idx_b shard 1 not found")
	}
}

func TestNewStore(t *testing.T) {
	s := NewStore(nil, "test_index", 5)
	if s == nil {
		t.Fatal("NewStore returned nil")
	}
	if s.table != "index_test_index_0005" {
		t.Errorf("table: got %q, want %q", s.table, "index_test_index_0005")
	}
}

func TestDefinition_Fields(t *testing.T) {
	def := Definition{
		Name:          "idx",
		SourceColumn:  "col",
		ShardKeyField: "field",
		Fields:        []string{"a", "b"},
	}

	if def.Name != "idx" {
		t.Error("Name mismatch")
	}
	if def.SourceColumn != "col" {
		t.Error("SourceColumn mismatch")
	}
	if def.ShardKeyField != "field" {
		t.Error("ShardKeyField mismatch")
	}
	if len(def.Fields) != 2 || def.Fields[0] != "a" || def.Fields[1] != "b" {
		t.Error("Fields mismatch")
	}
}
