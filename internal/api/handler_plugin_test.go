package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/ryanbastic/go-mezzanine/internal/index"
	"github.com/ryanbastic/go-mezzanine/internal/shard"
	"github.com/ryanbastic/go-mezzanine/internal/trigger"
)

func setupPluginTestServer() http.Handler {
	registry := trigger.NewPluginRegistry()
	return NewServer(testLogger(), shard.NewRouter(), index.NewRegistry(), registry, nil, 64)
}

func TestRegisterPlugin_Success(t *testing.T) {
	server := setupPluginTestServer()

	body := map[string]any{
		"name":               "test-plugin",
		"endpoint":           "http://localhost:9000/rpc",
		"subscribed_columns": []string{"profile", "settings"},
	}
	data, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/v1/plugins", bytes.NewReader(data))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("status: got %d, want %d\nbody: %s", w.Code, http.StatusCreated, w.Body.String())
	}

	var resp PluginResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Name != "test-plugin" {
		t.Errorf("Name: got %q", resp.Name)
	}
	if resp.Status != "active" {
		t.Errorf("Status: got %q", resp.Status)
	}
	if resp.ID == uuid.Nil {
		t.Error("expected non-nil ID")
	}
}

func TestRegisterPlugin_MissingName(t *testing.T) {
	server := setupPluginTestServer()

	body := map[string]any{
		"endpoint":           "http://localhost:9000/rpc",
		"subscribed_columns": []string{"profile"},
	}
	data, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/v1/plugins", bytes.NewReader(data))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.ServeHTTP(w, req)

	if w.Code < 400 || w.Code >= 500 {
		t.Errorf("status: got %d, want 4xx\nbody: %s", w.Code, w.Body.String())
	}
}

func TestRegisterPlugin_MissingColumns(t *testing.T) {
	server := setupPluginTestServer()

	body := map[string]any{
		"name":     "test-plugin",
		"endpoint": "http://localhost:9000/rpc",
	}
	data, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/v1/plugins", bytes.NewReader(data))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.ServeHTTP(w, req)

	if w.Code < 400 || w.Code >= 500 {
		t.Errorf("status: got %d, want 4xx\nbody: %s", w.Code, w.Body.String())
	}
}

func TestListPlugins_Empty(t *testing.T) {
	server := setupPluginTestServer()

	req := httptest.NewRequest(http.MethodGet, "/v1/plugins", nil)
	w := httptest.NewRecorder()

	server.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status: got %d, want %d", w.Code, http.StatusOK)
	}

	var resp []PluginResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(resp) != 0 {
		t.Errorf("expected empty list, got %d", len(resp))
	}
}

func TestListPlugins_AfterRegister(t *testing.T) {
	registry := trigger.NewPluginRegistry()
	server := NewServer(testLogger(), shard.NewRouter(), index.NewRegistry(), registry, nil, 64)

	// Register a plugin
	body := map[string]any{
		"name":               "test-plugin",
		"endpoint":           "http://localhost:9000/rpc",
		"subscribed_columns": []string{"profile"},
	}
	data, _ := json.Marshal(body)
	writeReq := httptest.NewRequest(http.MethodPost, "/v1/plugins", bytes.NewReader(data))
	writeReq.Header.Set("Content-Type", "application/json")
	writeW := httptest.NewRecorder()
	server.ServeHTTP(writeW, writeReq)

	if writeW.Code != http.StatusCreated {
		t.Fatalf("register status: got %d", writeW.Code)
	}

	// List
	req := httptest.NewRequest(http.MethodGet, "/v1/plugins", nil)
	w := httptest.NewRecorder()
	server.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status: got %d, want %d", w.Code, http.StatusOK)
	}

	var resp []PluginResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(resp) != 1 {
		t.Errorf("expected 1 plugin, got %d", len(resp))
	}
}

func TestGetPlugin_Success(t *testing.T) {
	registry := trigger.NewPluginRegistry()
	server := NewServer(testLogger(), shard.NewRouter(), index.NewRegistry(), registry, nil, 64)

	// Register
	p := &trigger.Plugin{
		Name:              "test",
		Endpoint:          "http://localhost:9000/rpc",
		SubscribedColumns: []string{"profile"},
	}
	registry.Register(p)

	req := httptest.NewRequest(http.MethodGet, "/v1/plugins/"+p.ID.String(), nil)
	w := httptest.NewRecorder()
	server.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status: got %d, want %d\nbody: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp PluginResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.ID != p.ID {
		t.Errorf("ID: got %s, want %s", resp.ID, p.ID)
	}
}

func TestGetPlugin_NotFound(t *testing.T) {
	server := setupPluginTestServer()

	req := httptest.NewRequest(http.MethodGet, "/v1/plugins/"+uuid.New().String(), nil)
	w := httptest.NewRecorder()
	server.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status: got %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestGetPlugin_InvalidID(t *testing.T) {
	server := setupPluginTestServer()

	req := httptest.NewRequest(http.MethodGet, "/v1/plugins/not-a-uuid", nil)
	w := httptest.NewRecorder()
	server.ServeHTTP(w, req)

	if w.Code < 400 || w.Code >= 500 {
		t.Errorf("status: got %d, want 4xx", w.Code)
	}
}

func TestDeletePlugin_Success(t *testing.T) {
	registry := trigger.NewPluginRegistry()
	server := NewServer(testLogger(), shard.NewRouter(), index.NewRegistry(), registry, nil, 64)

	p := &trigger.Plugin{
		Name:              "test",
		Endpoint:          "http://localhost:9000/rpc",
		SubscribedColumns: []string{"profile"},
	}
	registry.Register(p)

	req := httptest.NewRequest(http.MethodDelete, "/v1/plugins/"+p.ID.String(), nil)
	w := httptest.NewRecorder()
	server.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("status: got %d, want %d\nbody: %s", w.Code, http.StatusNoContent, w.Body.String())
	}

	// Verify deleted
	if len(registry.List()) != 0 {
		t.Error("expected registry to be empty after delete")
	}
}

func TestDeletePlugin_NotFound(t *testing.T) {
	server := setupPluginTestServer()

	req := httptest.NewRequest(http.MethodDelete, "/v1/plugins/"+uuid.New().String(), nil)
	w := httptest.NewRecorder()
	server.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status: got %d, want %d", w.Code, http.StatusNotFound)
	}
}
