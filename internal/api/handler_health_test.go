package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ryanbastic/go-mezzanine/internal/index"
	"github.com/ryanbastic/go-mezzanine/internal/shard"
	"github.com/ryanbastic/go-mezzanine/internal/trigger"
)

// --- Mock Pinger ---

type mockPinger struct {
	err error
}

func (m *mockPinger) Ping(_ context.Context) error {
	return m.err
}

// --- Livez ---

func TestLivez_ReturnsOK(t *testing.T) {
	server := NewServer(testLogger(), shard.NewRouter(), index.NewRegistry(), trigger.NewPluginRegistry(), nil, 64, nil)

	req := httptest.NewRequest(http.MethodGet, "/v1/livez", nil)
	w := httptest.NewRecorder()
	server.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status: got %d, want %d", w.Code, http.StatusOK)
	}

	var resp map[string]string
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp["status"] != "ok" {
		t.Errorf("status: got %q, want %q", resp["status"], "ok")
	}
}

// --- Readyz ---

func TestReadyz_NoBackends_ReturnsOK(t *testing.T) {
	server := NewServer(testLogger(), shard.NewRouter(), index.NewRegistry(), trigger.NewPluginRegistry(), nil, 64, nil)

	req := httptest.NewRequest(http.MethodGet, "/v1/readyz", nil)
	w := httptest.NewRecorder()
	server.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status: got %d, want %d", w.Code, http.StatusOK)
	}
}

func TestReadyz_AllHealthy(t *testing.T) {
	backends := map[string]Pinger{
		"pg1": &mockPinger{},
		"pg2": &mockPinger{},
	}
	server := NewServer(testLogger(), shard.NewRouter(), index.NewRegistry(), trigger.NewPluginRegistry(), nil, 64, backends)

	req := httptest.NewRequest(http.MethodGet, "/v1/readyz", nil)
	w := httptest.NewRecorder()
	server.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status: got %d, want %d\nbody: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp readyzResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Status != "ok" {
		t.Errorf("status: got %q, want %q", resp.Status, "ok")
	}
	if len(resp.Backends) != 2 {
		t.Fatalf("backends: got %d, want 2", len(resp.Backends))
	}
	for name, bs := range resp.Backends {
		if bs.Status != "ok" {
			t.Errorf("backend %s: got %q, want %q", name, bs.Status, "ok")
		}
	}
}

func TestReadyz_OneBackendDown(t *testing.T) {
	backends := map[string]Pinger{
		"pg1": &mockPinger{},
		"pg2": &mockPinger{err: errors.New("connection refused")},
	}
	server := NewServer(testLogger(), shard.NewRouter(), index.NewRegistry(), trigger.NewPluginRegistry(), nil, 64, backends)

	req := httptest.NewRequest(http.MethodGet, "/v1/readyz", nil)
	w := httptest.NewRecorder()
	server.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("status: got %d, want %d\nbody: %s", w.Code, http.StatusServiceUnavailable, w.Body.String())
	}

	var resp readyzResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Status != "unavailable" {
		t.Errorf("status: got %q, want %q", resp.Status, "unavailable")
	}
	if resp.Backends["pg1"].Status != "ok" {
		t.Errorf("pg1: got %q, want %q", resp.Backends["pg1"].Status, "ok")
	}
	if resp.Backends["pg2"].Status != "error" {
		t.Errorf("pg2: got %q, want %q", resp.Backends["pg2"].Status, "error")
	}
	if resp.Backends["pg2"].Error != "connection refused" {
		t.Errorf("pg2 error: got %q", resp.Backends["pg2"].Error)
	}
}

// --- /v1/health backwards compat ---

func TestHealth_BackwardsCompat_BehavesAsReadyz(t *testing.T) {
	backends := map[string]Pinger{
		"pg1": &mockPinger{},
	}
	server := NewServer(testLogger(), shard.NewRouter(), index.NewRegistry(), trigger.NewPluginRegistry(), nil, 64, backends)

	req := httptest.NewRequest(http.MethodGet, "/v1/health", nil)
	w := httptest.NewRecorder()
	server.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status: got %d, want %d", w.Code, http.StatusOK)
	}

	var resp readyzResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Status != "ok" {
		t.Errorf("status: got %q, want %q", resp.Status, "ok")
	}
	if resp.Backends["pg1"].Status != "ok" {
		t.Errorf("pg1: got %q, want %q", resp.Backends["pg1"].Status, "ok")
	}
}
