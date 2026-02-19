package trigger

import (
	"context"
	"testing"

	"github.com/google/uuid"
)

func TestPluginRegistry_RegisterAndGet(t *testing.T) {
	r := NewPluginRegistry()
	p := &Plugin{
		Name:              "test-plugin",
		Endpoint:          "http://localhost:9000/rpc",
		SubscribedColumns: []string{"profile", "settings"},
	}
	if err := r.Register(context.Background(), p); err != nil {
		t.Fatalf("Register: %v", err)
	}

	if p.ID == uuid.Nil {
		t.Fatal("expected ID to be assigned")
	}
	if p.CreatedAt.IsZero() {
		t.Fatal("expected CreatedAt to be set")
	}
	if p.Status != PluginStatusActive {
		t.Errorf("status: got %q, want %q", p.Status, PluginStatusActive)
	}

	got, err := r.Get(p.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Name != "test-plugin" {
		t.Errorf("Name: got %q", got.Name)
	}
}

func TestPluginRegistry_GetNotFound(t *testing.T) {
	r := NewPluginRegistry()
	_, err := r.Get(uuid.New())
	if err == nil {
		t.Fatal("expected error for missing plugin")
	}
}

func TestPluginRegistry_RegisterDuplicate(t *testing.T) {
	r := NewPluginRegistry()
	p := &Plugin{
		Name:              "dup-plugin",
		Endpoint:          "http://localhost:9000/rpc",
		SubscribedColumns: []string{"profile"},
	}
	if err := r.Register(context.Background(), p); err != nil {
		t.Fatalf("first Register: %v", err)
	}

	dup := &Plugin{
		Name:              "dup-plugin",
		Endpoint:          "http://localhost:9001/rpc",
		SubscribedColumns: []string{"settings"},
	}
	if err := r.Register(context.Background(), dup); err == nil {
		t.Fatal("expected error for duplicate plugin name")
	}

	if len(r.List()) != 1 {
		t.Errorf("expected 1 plugin, got %d", len(r.List()))
	}
}

func TestPluginRegistry_List(t *testing.T) {
	r := NewPluginRegistry()
	r.Register(context.Background(), &Plugin{Name: "a", Endpoint: "http://a/rpc", SubscribedColumns: []string{"x"}}) //nolint:errcheck
	r.Register(context.Background(), &Plugin{Name: "b", Endpoint: "http://b/rpc", SubscribedColumns: []string{"y"}}) //nolint:errcheck

	list := r.List()
	if len(list) != 2 {
		t.Errorf("List: got %d, want 2", len(list))
	}
}

func TestPluginRegistry_Delete(t *testing.T) {
	r := NewPluginRegistry()
	p := &Plugin{Name: "del", Endpoint: "http://del/rpc", SubscribedColumns: []string{"x"}}
	if err := r.Register(context.Background(), p); err != nil {
		t.Fatalf("Register: %v", err)
	}

	if err := r.Delete(p.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if len(r.List()) != 0 {
		t.Error("expected empty list after delete")
	}
}

func TestPluginRegistry_DeleteNotFound(t *testing.T) {
	r := NewPluginRegistry()
	if err := r.Delete(uuid.New()); err == nil {
		t.Fatal("expected error for missing plugin")
	}
}

func TestPluginRegistry_ForColumn(t *testing.T) {
	r := NewPluginRegistry()
	r.Register(context.Background(), &Plugin{Name: "a", Endpoint: "http://a/rpc", SubscribedColumns: []string{"profile", "settings"}})                  //nolint:errcheck
	r.Register(context.Background(), &Plugin{Name: "b", Endpoint: "http://b/rpc", SubscribedColumns: []string{"profile"}})                              //nolint:errcheck
	r.Register(context.Background(), &Plugin{Name: "c", Endpoint: "http://c/rpc", SubscribedColumns: []string{"orders"}, Status: PluginStatusInactive}) //nolint:errcheck

	got := r.ForColumn("profile")
	if len(got) != 2 {
		t.Errorf("ForColumn(profile): got %d, want 2", len(got))
	}

	got = r.ForColumn("orders")
	if len(got) != 0 {
		t.Errorf("ForColumn(orders): got %d, want 0 (inactive)", len(got))
	}

	got = r.ForColumn("nonexistent")
	if len(got) != 0 {
		t.Errorf("ForColumn(nonexistent): got %d, want 0", len(got))
	}
}

func TestPluginRegistry_Columns(t *testing.T) {
	r := NewPluginRegistry()
	r.Register(context.Background(), &Plugin{Name: "a", Endpoint: "http://a/rpc", SubscribedColumns: []string{"profile", "settings"}})                  //nolint:errcheck
	r.Register(context.Background(), &Plugin{Name: "b", Endpoint: "http://b/rpc", SubscribedColumns: []string{"profile", "orders"}})                    //nolint:errcheck
	r.Register(context.Background(), &Plugin{Name: "c", Endpoint: "http://c/rpc", SubscribedColumns: []string{"hidden"}, Status: PluginStatusInactive}) //nolint:errcheck

	cols := r.Columns()
	colSet := make(map[string]bool)
	for _, c := range cols {
		colSet[c] = true
	}

	if len(cols) != 3 {
		t.Errorf("Columns: got %d, want 3", len(cols))
	}
	for _, expected := range []string{"profile", "settings", "orders"} {
		if !colSet[expected] {
			t.Errorf("Columns: missing %q", expected)
		}
	}
	if colSet["hidden"] {
		t.Error("Columns: should not include inactive plugin columns")
	}
}
