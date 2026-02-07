package index

import (
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/ryanbastic/go-mezzanine/internal/cell"
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

// --- extractUUID Tests ---

func TestExtractUUID_Valid(t *testing.T) {
	id := uuid.New()
	body := []byte(`{"user_id":"` + id.String() + `"}`)

	got, err := extractUUID(json.RawMessage(body), "user_id")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != id {
		t.Errorf("got %s, want %s", got, id)
	}
}

func TestExtractUUID_MissingField(t *testing.T) {
	body := []byte(`{"other":"value"}`)
	_, err := extractUUID(json.RawMessage(body), "user_id")
	if err == nil {
		t.Fatal("expected error for missing field")
	}
}

func TestExtractUUID_NonStringField(t *testing.T) {
	body := []byte(`{"user_id":12345}`)
	_, err := extractUUID(json.RawMessage(body), "user_id")
	if err == nil {
		t.Fatal("expected error for non-string field")
	}
}

func TestExtractUUID_InvalidUUID(t *testing.T) {
	body := []byte(`{"user_id":"not-a-uuid"}`)
	_, err := extractUUID(json.RawMessage(body), "user_id")
	if err == nil {
		t.Fatal("expected error for invalid UUID")
	}
}

// --- extractFields Tests ---

func TestExtractFields_Subset(t *testing.T) {
	body := []byte(`{"email":"a@b.com","name":"Alice","age":30}`)
	got, err := extractFields(json.RawMessage(body), []string{"email", "name"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var m map[string]json.RawMessage
	if err := json.Unmarshal(got, &m); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}
	if len(m) != 2 {
		t.Errorf("got %d keys, want 2", len(m))
	}
	if _, ok := m["email"]; !ok {
		t.Error("missing email")
	}
	if _, ok := m["name"]; !ok {
		t.Error("missing name")
	}
	if _, ok := m["age"]; ok {
		t.Error("age should not be included")
	}
}

func TestExtractFields_MissingFieldsSkipped(t *testing.T) {
	body := []byte(`{"email":"a@b.com"}`)
	got, err := extractFields(json.RawMessage(body), []string{"email", "nonexistent"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var m map[string]json.RawMessage
	if err := json.Unmarshal(got, &m); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}
	if len(m) != 1 {
		t.Errorf("got %d keys, want 1", len(m))
	}
}

func TestExtractFields_EmptyList(t *testing.T) {
	body := []byte(`{"email":"a@b.com"}`)
	got, err := extractFields(json.RawMessage(body), []string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var m map[string]json.RawMessage
	if err := json.Unmarshal(got, &m); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}
	if len(m) != 0 {
		t.Errorf("got %d keys, want 0", len(m))
	}
}

// --- ForColumn Tests ---

func TestRegistry_ForColumn_Matches(t *testing.T) {
	r := NewRegistry()
	r.Register(nil, Definition{Name: "idx_a", SourceColumn: "profile"}, 2)
	r.Register(nil, Definition{Name: "idx_b", SourceColumn: "profile"}, 2)
	r.Register(nil, Definition{Name: "idx_c", SourceColumn: "settings"}, 2)

	defs := r.ForColumn("profile")
	if len(defs) != 2 {
		t.Errorf("got %d definitions, want 2", len(defs))
	}
}

func TestRegistry_ForColumn_NoMatches(t *testing.T) {
	r := NewRegistry()
	r.Register(nil, Definition{Name: "idx_a", SourceColumn: "profile"}, 2)

	defs := r.ForColumn("nonexistent")
	if len(defs) != 0 {
		t.Errorf("got %d definitions, want 0", len(defs))
	}
}

// --- IndexCell Tests ---

func TestRegistry_IndexCell_NoMatchingDefs(t *testing.T) {
	r := NewRegistry()

	c := &cell.Cell{
		RowKey:     uuid.New(),
		ColumnName: "unmatched",
		Body:       json.RawMessage(`{}`),
	}

	// No definitions registered, so nothing to index — should succeed.
	if err := r.IndexCell(t.Context(), c, 4); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRegistry_IndexCell_ExtractUUIDError(t *testing.T) {
	r := NewRegistry()
	r.Register(nil, Definition{
		Name:          "idx",
		SourceColumn:  "profile",
		ShardKeyField: "user_id",
		Fields:        []string{"email"},
	}, 4)

	c := &cell.Cell{
		RowKey:     uuid.New(),
		ColumnName: "profile",
		Body:       json.RawMessage(`{"email":"a@b.com"}`), // missing user_id
	}

	err := r.IndexCell(t.Context(), c, 4)
	if err == nil {
		t.Fatal("expected error for missing shard key field")
	}
}
