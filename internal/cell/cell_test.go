package cell

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestCellRef_JSONRoundTrip(t *testing.T) {
	ref := CellRef{
		RowKey:     uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"),
		ColumnName: "profile",
		RefKey:     42,
	}

	data, err := json.Marshal(ref)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var got CellRef
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if got != ref {
		t.Errorf("got %+v, want %+v", got, ref)
	}
}

func TestCellRef_JSONFields(t *testing.T) {
	ref := CellRef{
		RowKey:     uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"),
		ColumnName: "profile",
		RefKey:     1,
	}

	data, err := json.Marshal(ref)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("unmarshal to map: %v", err)
	}

	if _, ok := m["row_key"]; !ok {
		t.Error("expected row_key JSON field")
	}
	if _, ok := m["column_name"]; !ok {
		t.Error("expected column_name JSON field")
	}
	if _, ok := m["ref_key"]; !ok {
		t.Error("expected ref_key JSON field")
	}
}

func TestCell_JSONRoundTrip(t *testing.T) {
	now := time.Now().Truncate(time.Millisecond)
	c := Cell{
		AddedID:    100,
		RowKey:     uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"),
		ColumnName: "profile",
		RefKey:     1,
		Body:       json.RawMessage(`{"name":"test"}`),
		CreatedAt:  now,
	}

	data, err := json.Marshal(c)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var got Cell
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if got.AddedID != c.AddedID {
		t.Errorf("AddedID: got %d, want %d", got.AddedID, c.AddedID)
	}
	if got.RowKey != c.RowKey {
		t.Errorf("RowKey: got %s, want %s", got.RowKey, c.RowKey)
	}
	if got.ColumnName != c.ColumnName {
		t.Errorf("ColumnName: got %s, want %s", got.ColumnName, c.ColumnName)
	}
	if got.RefKey != c.RefKey {
		t.Errorf("RefKey: got %d, want %d", got.RefKey, c.RefKey)
	}
	if string(got.Body) != string(c.Body) {
		t.Errorf("Body: got %s, want %s", got.Body, c.Body)
	}
}

func TestCell_JSONFields(t *testing.T) {
	c := Cell{
		AddedID:    1,
		RowKey:     uuid.New(),
		ColumnName: "col",
		RefKey:     1,
		Body:       json.RawMessage(`{}`),
		CreatedAt:  time.Now(),
	}

	data, err := json.Marshal(c)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("unmarshal to map: %v", err)
	}

	expected := []string{"added_id", "row_key", "column_name", "ref_key", "body", "created_at"}
	for _, key := range expected {
		if _, ok := m[key]; !ok {
			t.Errorf("expected JSON field %q", key)
		}
	}
}

func TestWriteCellRequest_JSONRoundTrip(t *testing.T) {
	req := WriteCellRequest{
		RowKey:     uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"),
		ColumnName: "profile",
		RefKey:     1,
		Body:       json.RawMessage(`{"email":"test@example.com"}`),
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var got WriteCellRequest
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if got.RowKey != req.RowKey {
		t.Errorf("RowKey: got %s, want %s", got.RowKey, req.RowKey)
	}
	if got.ColumnName != req.ColumnName {
		t.Errorf("ColumnName: got %s, want %s", got.ColumnName, req.ColumnName)
	}
	if got.RefKey != req.RefKey {
		t.Errorf("RefKey: got %d, want %d", got.RefKey, req.RefKey)
	}
	if string(got.Body) != string(req.Body) {
		t.Errorf("Body: got %s, want %s", got.Body, req.Body)
	}
}

func TestWriteCellRequest_FromJSON(t *testing.T) {
	raw := `{
		"row_key": "550e8400-e29b-41d4-a716-446655440000",
		"column_name": "profile",
		"ref_key": 7,
		"body": {"key": "value"}
	}`

	var req WriteCellRequest
	if err := json.Unmarshal([]byte(raw), &req); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if req.RowKey != uuid.MustParse("550e8400-e29b-41d4-a716-446655440000") {
		t.Errorf("unexpected RowKey: %s", req.RowKey)
	}
	if req.ColumnName != "profile" {
		t.Errorf("unexpected ColumnName: %s", req.ColumnName)
	}
	if req.RefKey != 7 {
		t.Errorf("unexpected RefKey: %d", req.RefKey)
	}
	if req.Body == nil {
		t.Error("expected non-nil Body")
	}
}

func TestCell_BodyPreservesRawJSON(t *testing.T) {
	body := json.RawMessage(`{"nested":{"deep":true},"array":[1,2,3]}`)
	c := Cell{
		AddedID:    1,
		RowKey:     uuid.New(),
		ColumnName: "test",
		RefKey:     1,
		Body:       body,
		CreatedAt:  time.Now(),
	}

	data, err := json.Marshal(c)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var got Cell
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	// Verify the nested JSON structure is preserved
	var parsed map[string]any
	if err := json.Unmarshal(got.Body, &parsed); err != nil {
		t.Fatalf("parse body: %v", err)
	}
	if _, ok := parsed["nested"]; !ok {
		t.Error("expected nested key in body")
	}
	if _, ok := parsed["array"]; !ok {
		t.Error("expected array key in body")
	}
}
