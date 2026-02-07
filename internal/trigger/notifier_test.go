package trigger

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/ryanbastic/go-mezzanine/internal/cell"
)

func TestNotifier_DispatchesToSubscribedPlugins(t *testing.T) {
	var received atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		received.Add(1)
		var req JSONRPCRequest
		json.NewDecoder(r.Body).Decode(&req)
		resp := JSONRPCResponse{JSONRPC: "2.0", Result: json.RawMessage(`"ok"`), ID: req.ID}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	registry := NewPluginRegistry()
	registry.Register(&Plugin{
		Name:              "plugin-a",
		Endpoint:          srv.URL,
		SubscribedColumns: []string{"profile"},
	})
	registry.Register(&Plugin{
		Name:              "plugin-b",
		Endpoint:          srv.URL,
		SubscribedColumns: []string{"profile", "settings"},
	})

	rpcClient := NewRPCClient(0, time.Millisecond, 5*time.Second)
	notifier := NewNotifier(registry, rpcClient, slog.New(slog.DiscardHandler))

	c := &cell.Cell{
		AddedID:    1,
		RowKey:     uuid.New(),
		ColumnName: "profile",
		RefKey:     1,
		Body:       json.RawMessage(`{"v":1}`),
		CreatedAt:  time.Now(),
	}

	notifier.NotifyCell(0, c)

	// Wait for goroutines to complete
	time.Sleep(200 * time.Millisecond)

	if received.Load() != 2 {
		t.Errorf("received: got %d, want 2", received.Load())
	}
}

func TestNotifier_SkipsUnsubscribedPlugins(t *testing.T) {
	var received atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		received.Add(1)
		var req JSONRPCRequest
		json.NewDecoder(r.Body).Decode(&req)
		resp := JSONRPCResponse{JSONRPC: "2.0", Result: json.RawMessage(`"ok"`), ID: req.ID}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	registry := NewPluginRegistry()
	registry.Register(&Plugin{
		Name:              "settings-only",
		Endpoint:          srv.URL,
		SubscribedColumns: []string{"settings"},
	})

	rpcClient := NewRPCClient(0, time.Millisecond, 5*time.Second)
	notifier := NewNotifier(registry, rpcClient, slog.New(slog.DiscardHandler))

	c := &cell.Cell{
		AddedID:    1,
		RowKey:     uuid.New(),
		ColumnName: "profile",
		RefKey:     1,
		Body:       json.RawMessage(`{"v":1}`),
		CreatedAt:  time.Now(),
	}

	notifier.NotifyCell(0, c)

	time.Sleep(100 * time.Millisecond)

	if received.Load() != 0 {
		t.Errorf("received: got %d, want 0", received.Load())
	}
}

func TestNotifier_LogsRPCErrors(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	var mu sync.Mutex
	var logged bool

	handler := slog.NewTextHandler(writerFunc(func(p []byte) (int, error) {
		mu.Lock()
		defer mu.Unlock()
		logged = true
		return len(p), nil
	}), nil)

	registry := NewPluginRegistry()
	registry.Register(&Plugin{
		Name:              "failing",
		Endpoint:          srv.URL,
		SubscribedColumns: []string{"profile"},
	})

	rpcClient := NewRPCClient(0, time.Millisecond, 5*time.Second)
	notifier := NewNotifier(registry, rpcClient, slog.New(handler))

	c := &cell.Cell{
		AddedID:    1,
		RowKey:     uuid.New(),
		ColumnName: "profile",
		RefKey:     1,
		Body:       json.RawMessage(`{"v":1}`),
		CreatedAt:  time.Now(),
	}

	notifier.NotifyCell(0, c)

	time.Sleep(200 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()
	if !logged {
		t.Error("expected error to be logged")
	}
}

func TestNotifier_NoPlugins(t *testing.T) {
	registry := NewPluginRegistry()
	rpcClient := NewRPCClient(0, time.Millisecond, 5*time.Second)
	notifier := NewNotifier(registry, rpcClient, slog.New(slog.DiscardHandler))

	c := &cell.Cell{
		AddedID:    1,
		RowKey:     uuid.New(),
		ColumnName: "profile",
		RefKey:     1,
		Body:       json.RawMessage(`{"v":1}`),
		CreatedAt:  time.Now(),
	}

	// Should not panic
	notifier.NotifyCell(0, c)
}

// writerFunc adapts a function to the io.Writer interface.
type writerFunc func(p []byte) (int, error)

func (f writerFunc) Write(p []byte) (int, error) {
	return f(p)
}
