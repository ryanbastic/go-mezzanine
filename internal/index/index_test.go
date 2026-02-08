package index

import (
	"encoding/json"
	"strings"
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
		ShardKeyField: "email",
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
		ShardKeyField: "email",
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
	if got.ShardKeyField != "email" {
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

func TestDefinition_UniqueFields(t *testing.T) {
	def := Definition{
		Name:          "user_by_email",
		SourceColumn:  "profile",
		ShardKeyField: "email",
		Fields:        []string{"email", "display_name"},
		UniqueFields:  []string{"email"},
	}

	if len(def.UniqueFields) != 1 || def.UniqueFields[0] != "email" {
		t.Errorf("UniqueFields: got %v", def.UniqueFields)
	}
}

func TestBuildTableDDL_NoUniqueFields(t *testing.T) {
	ddl := buildTableDDL("index_test_0000", nil)
	if !strings.Contains(ddl, "CREATE TABLE IF NOT EXISTS index_test_0000") {
		t.Error("missing CREATE TABLE")
	}
	if !strings.Contains(ddl, "idx_index_test_0000_shard_key") {
		t.Error("missing shard_key index")
	}
	if strings.Contains(ddl, "UNIQUE") {
		t.Error("should not contain UNIQUE when no unique fields")
	}
}

func TestBuildTableDDL_WithUniqueFields(t *testing.T) {
	ddl := buildTableDDL("index_user_by_email_0000", []string{"email"})
	if !strings.Contains(ddl, "CREATE TABLE IF NOT EXISTS index_user_by_email_0000") {
		t.Error("missing CREATE TABLE")
	}
	if !strings.Contains(ddl, "CREATE UNIQUE INDEX IF NOT EXISTS idx_index_user_by_email_0000_email") {
		t.Error("missing unique index on email")
	}
	if !strings.Contains(ddl, "(body->>'email')") {
		t.Error("missing body->>'email' expression")
	}
}

func TestBuildTableDDL_MultipleUniqueFields(t *testing.T) {
	ddl := buildTableDDL("index_test_0000", []string{"email", "username"})
	if !strings.Contains(ddl, "idx_index_test_0000_email") {
		t.Error("missing unique index on email")
	}
	if !strings.Contains(ddl, "idx_index_test_0000_username") {
		t.Error("missing unique index on username")
	}
}

// --- extractString Tests ---

func TestExtractString_Valid(t *testing.T) {
	body := []byte(`{"email":"alice@example.com"}`)

	got, err := extractString(json.RawMessage(body), "email")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "alice@example.com" {
		t.Errorf("got %s, want alice@example.com", got)
	}
}

func TestExtractString_UUID(t *testing.T) {
	id := uuid.New()
	body := []byte(`{"user_id":"` + id.String() + `"}`)

	got, err := extractString(json.RawMessage(body), "user_id")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != id.String() {
		t.Errorf("got %s, want %s", got, id.String())
	}
}

func TestExtractString_MissingField(t *testing.T) {
	body := []byte(`{"other":"value"}`)
	_, err := extractString(json.RawMessage(body), "email")
	if err == nil {
		t.Fatal("expected error for missing field")
	}
}

func TestExtractString_NonStringField(t *testing.T) {
	body := []byte(`{"email":12345}`)
	_, err := extractString(json.RawMessage(body), "email")
	if err == nil {
		t.Fatal("expected error for non-string field")
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

// --- RegisterRange Tests ---

func TestRegistry_RegisterRange_SingleRange(t *testing.T) {
	r := NewRegistry()
	def := Definition{
		Name:          "user_by_email",
		SourceColumn:  "profile",
		ShardKeyField: "email",
		Fields:        []string{"email"},
	}

	r.RegisterRange(nil, def, 0, 3)

	for i := 0; i <= 3; i++ {
		store, ok := r.StoreFor("user_by_email", shard.ID(i))
		if !ok {
			t.Errorf("StoreFor shard %d: not found", i)
		}
		if store == nil {
			t.Errorf("StoreFor shard %d: nil store", i)
		}
	}

	// Shard 4 should not exist
	if _, ok := r.StoreFor("user_by_email", shard.ID(4)); ok {
		t.Error("shard 4 should not exist")
	}
}

func TestRegistry_RegisterRange_MultipleRanges(t *testing.T) {
	r := NewRegistry()
	def := Definition{
		Name:          "user_by_email",
		SourceColumn:  "profile",
		ShardKeyField: "email",
		Fields:        []string{"email"},
	}

	// Simulate two backends
	r.RegisterRange(nil, def, 0, 1)
	r.RegisterRange(nil, def, 2, 3)

	for i := 0; i <= 3; i++ {
		store, ok := r.StoreFor("user_by_email", shard.ID(i))
		if !ok {
			t.Errorf("StoreFor shard %d: not found", i)
		}
		if store == nil {
			t.Errorf("StoreFor shard %d: nil store", i)
		}
	}
}

func TestRegistry_RegisterRange_SingleShard(t *testing.T) {
	r := NewRegistry()
	def := Definition{Name: "test_idx"}
	r.RegisterRange(nil, def, 5, 5)

	if _, ok := r.StoreFor("test_idx", shard.ID(5)); !ok {
		t.Error("shard 5 not found")
	}
	if _, ok := r.StoreFor("test_idx", shard.ID(4)); ok {
		t.Error("shard 4 should not exist")
	}
	if _, ok := r.StoreFor("test_idx", shard.ID(6)); ok {
		t.Error("shard 6 should not exist")
	}
}

func TestRegistry_RegisterRange_DefinitionPreserved(t *testing.T) {
	r := NewRegistry()
	def := Definition{
		Name:          "user_by_email",
		SourceColumn:  "profile",
		ShardKeyField: "email",
		Fields:        []string{"email", "name"},
	}
	r.RegisterRange(nil, def, 0, 1)

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
	if got.ShardKeyField != "email" {
		t.Errorf("ShardKeyField: got %q", got.ShardKeyField)
	}
	if len(got.Fields) != 2 {
		t.Errorf("Fields: got %d", len(got.Fields))
	}
}

func TestRegistry_UserByEmail_IndexCell_FieldExtraction(t *testing.T) {
	r := NewRegistry()
	def := Definition{
		Name:          "user_by_email",
		SourceColumn:  "profile",
		ShardKeyField: "email",
		Fields:        []string{"email", "display_name"},
		UniqueFields:  []string{"email"},
	}
	r.Register(nil, def, 4)

	rowKey := uuid.New()

	// Verify that ForColumn finds the user_by_email definition.
	defs := r.ForColumn("profile")
	if len(defs) != 1 {
		t.Fatalf("ForColumn: got %d, want 1", len(defs))
	}
	if defs[0].Name != "user_by_email" {
		t.Errorf("Name: got %q", defs[0].Name)
	}

	// Verify field extraction matches the definition.
	body := json.RawMessage(`{
		"org_id": "some-org",
		"email": "alice@example.com",
		"display_name": "Alice Smith",
		"internal_notes": "should not appear"
	}`)

	gotEmail, err := extractString(body, "email")
	if err != nil {
		t.Fatalf("extractString: %v", err)
	}
	if gotEmail != "alice@example.com" {
		t.Errorf("shard key: got %s, want alice@example.com", gotEmail)
	}

	gotBody, err := extractFields(body, def.Fields)
	if err != nil {
		t.Fatalf("extractFields: %v", err)
	}

	var m map[string]json.RawMessage
	if err := json.Unmarshal(gotBody, &m); err != nil {
		t.Fatalf("unmarshal body: %v", err)
	}
	if len(m) != 2 {
		t.Errorf("body keys: got %d, want 2", len(m))
	}
	if string(m["email"]) != `"alice@example.com"` {
		t.Errorf("email: got %s", string(m["email"]))
	}
	if string(m["display_name"]) != `"Alice Smith"` {
		t.Errorf("display_name: got %s", string(m["display_name"]))
	}
	if _, ok := m["internal_notes"]; ok {
		t.Error("internal_notes should not be extracted")
	}

	// Verify shard routing for the index entry.
	shardID := shard.ForKey("alice@example.com", 4)
	store, ok := r.StoreFor("user_by_email", shardID)
	if !ok {
		t.Fatalf("StoreFor shard %d: not found", shardID)
	}
	if store == nil {
		t.Fatal("store is nil")
	}

	// Verify the definition roundtrip.
	gotDef, ok := r.GetDefinition("user_by_email")
	if !ok {
		t.Fatal("GetDefinition: not found")
	}
	if len(gotDef.UniqueFields) != 1 || gotDef.UniqueFields[0] != "email" {
		t.Errorf("UniqueFields: got %v", gotDef.UniqueFields)
	}

	_ = rowKey // used by callers to build cells
}

func TestRegistry_UserByEmail_NonProfileColumn_Skipped(t *testing.T) {
	r := NewRegistry()
	r.Register(nil, Definition{
		Name:          "user_by_email",
		SourceColumn:  "profile",
		ShardKeyField: "email",
		Fields:        []string{"email"},
		UniqueFields:  []string{"email"},
	}, 4)

	// Write to a different column — index should be skipped entirely.
	c := &cell.Cell{
		RowKey:     uuid.New(),
		ColumnName: "settings",
		Body:       json.RawMessage(`{"theme":"dark"}`),
	}

	if err := r.IndexCell(t.Context(), c, 4); err != nil {
		t.Fatalf("IndexCell for non-matching column should succeed: %v", err)
	}
}

func TestRegistry_IndexCell_ExtractStringError(t *testing.T) {
	r := NewRegistry()
	r.Register(nil, Definition{
		Name:          "idx",
		SourceColumn:  "profile",
		ShardKeyField: "email",
		Fields:        []string{"email"},
	}, 4)

	c := &cell.Cell{
		RowKey:     uuid.New(),
		ColumnName: "profile",
		Body:       json.RawMessage(`{"name":"Alice"}`), // missing email
	}

	err := r.IndexCell(t.Context(), c, 4)
	if err == nil {
		t.Fatal("expected error for missing shard key field")
	}
}
