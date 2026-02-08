package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ryanbastic/go-mezzanine/internal/index"
	"github.com/ryanbastic/go-mezzanine/internal/shard"
	"github.com/ryanbastic/go-mezzanine/internal/trigger"
)

func TestGetShardCount(t *testing.T) {
	const numShards = 16
	server := NewServer(testLogger(), shard.NewRouter(), index.NewRegistry(), trigger.NewPluginRegistry(), nil, numShards, nil)

	req := httptest.NewRequest(http.MethodGet, "/v1/shards/count", nil)
	w := httptest.NewRecorder()

	server.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status: got %d, want %d", w.Code, http.StatusOK)
	}

	var resp ShardCountResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if resp.NumShards != numShards {
		t.Errorf("num_shards: got %d, want %d", resp.NumShards, numShards)
	}
}
