package trigger

import (
	"context"
	"sort"
	"testing"

	"github.com/ryanbastic/go-mezzanine/internal/cell"
)

func TestNewRegistry(t *testing.T) {
	r := NewRegistry()
	if r == nil {
		t.Fatal("NewRegistry returned nil")
	}
}

func TestRegistry_Register_SingleHandler(t *testing.T) {
	r := NewRegistry()
	called := false
	r.Register("profile", func(ctx context.Context, c cell.Cell) error {
		called = true
		return nil
	})

	handlers := r.HandlersFor("profile")
	if len(handlers) != 1 {
		t.Fatalf("expected 1 handler, got %d", len(handlers))
	}

	// Verify the handler works
	handlers[0](context.Background(), cell.Cell{})
	if !called {
		t.Error("handler was not called")
	}
}

func TestRegistry_Register_MultipleHandlers_SameColumn(t *testing.T) {
	r := NewRegistry()
	r.Register("profile", func(ctx context.Context, c cell.Cell) error { return nil })
	r.Register("profile", func(ctx context.Context, c cell.Cell) error { return nil })
	r.Register("profile", func(ctx context.Context, c cell.Cell) error { return nil })

	handlers := r.HandlersFor("profile")
	if len(handlers) != 3 {
		t.Errorf("expected 3 handlers, got %d", len(handlers))
	}
}

func TestRegistry_Register_DifferentColumns(t *testing.T) {
	r := NewRegistry()
	r.Register("profile", func(ctx context.Context, c cell.Cell) error { return nil })
	r.Register("events", func(ctx context.Context, c cell.Cell) error { return nil })
	r.Register("metrics", func(ctx context.Context, c cell.Cell) error { return nil })

	if len(r.HandlersFor("profile")) != 1 {
		t.Error("expected 1 handler for profile")
	}
	if len(r.HandlersFor("events")) != 1 {
		t.Error("expected 1 handler for events")
	}
	if len(r.HandlersFor("metrics")) != 1 {
		t.Error("expected 1 handler for metrics")
	}
}

func TestRegistry_HandlersFor_NonExistent(t *testing.T) {
	r := NewRegistry()
	handlers := r.HandlersFor("nonexistent")
	if handlers != nil {
		t.Errorf("expected nil for nonexistent column, got %v", handlers)
	}
}

func TestRegistry_Columns_Empty(t *testing.T) {
	r := NewRegistry()
	cols := r.Columns()
	if len(cols) != 0 {
		t.Errorf("expected 0 columns, got %d", len(cols))
	}
}

func TestRegistry_Columns(t *testing.T) {
	r := NewRegistry()
	r.Register("b_col", func(ctx context.Context, c cell.Cell) error { return nil })
	r.Register("a_col", func(ctx context.Context, c cell.Cell) error { return nil })
	r.Register("c_col", func(ctx context.Context, c cell.Cell) error { return nil })

	cols := r.Columns()
	if len(cols) != 3 {
		t.Fatalf("expected 3 columns, got %d", len(cols))
	}

	sort.Strings(cols)
	expected := []string{"a_col", "b_col", "c_col"}
	for i, want := range expected {
		if cols[i] != want {
			t.Errorf("columns[%d] = %q, want %q", i, cols[i], want)
		}
	}
}

func TestRegistry_Columns_NoDuplicates(t *testing.T) {
	r := NewRegistry()
	r.Register("same", func(ctx context.Context, c cell.Cell) error { return nil })
	r.Register("same", func(ctx context.Context, c cell.Cell) error { return nil })

	cols := r.Columns()
	if len(cols) != 1 {
		t.Errorf("expected 1 unique column, got %d", len(cols))
	}
}

func TestRegistry_HandlerExecutionOrder(t *testing.T) {
	r := NewRegistry()
	var order []int

	r.Register("col", func(ctx context.Context, c cell.Cell) error {
		order = append(order, 1)
		return nil
	})
	r.Register("col", func(ctx context.Context, c cell.Cell) error {
		order = append(order, 2)
		return nil
	})
	r.Register("col", func(ctx context.Context, c cell.Cell) error {
		order = append(order, 3)
		return nil
	})

	for _, h := range r.HandlersFor("col") {
		h(context.Background(), cell.Cell{})
	}

	if len(order) != 3 {
		t.Fatalf("expected 3 calls, got %d", len(order))
	}
	for i, want := range []int{1, 2, 3} {
		if order[i] != want {
			t.Errorf("order[%d] = %d, want %d", i, order[i], want)
		}
	}
}
