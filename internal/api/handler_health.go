package api

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"sync"
	"time"
)

// Pinger is satisfied by *pgxpool.Pool.
type Pinger interface {
	Ping(ctx context.Context) error
}

// HealthHandler serves liveness and readiness probes.
type HealthHandler struct {
	backends map[string]Pinger
	logger   *slog.Logger
}

func NewHealthHandler(backends map[string]Pinger, logger *slog.Logger) *HealthHandler {
	return &HealthHandler{backends: backends, logger: logger}
}

type backendStatus struct {
	Status    string `json:"status"`
	LatencyMs int64  `json:"latency_ms,omitempty"`
	Error     string `json:"error,omitempty"`
}

type readyzResponse struct {
	Status   string                    `json:"status"`
	Backends map[string]backendStatus  `json:"backends,omitempty"`
}

// Livez is a simple liveness probe — if the process can serve HTTP, it's alive.
func (h *HealthHandler) Livez(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// Readyz checks all database backends concurrently and reports per-backend status.
func (h *HealthHandler) Readyz(w http.ResponseWriter, r *http.Request) {
	if len(h.backends) == 0 {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(readyzResponse{Status: "ok"})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	type result struct {
		name   string
		status backendStatus
	}

	var (
		wg      sync.WaitGroup
		results = make(chan result, len(h.backends))
	)

	for name, p := range h.backends {
		wg.Add(1)
		go func(name string, p Pinger) {
			defer wg.Done()
			start := time.Now()
			err := p.Ping(ctx)
			elapsed := time.Since(start)
			if err != nil {
				results <- result{name: name, status: backendStatus{
					Status:    "error",
					LatencyMs: elapsed.Milliseconds(),
					Error:     err.Error(),
				}}
				return
			}
			results <- result{name: name, status: backendStatus{
				Status:    "ok",
				LatencyMs: elapsed.Milliseconds(),
			}}
		}(name, p)
	}

	wg.Wait()
	close(results)

	resp := readyzResponse{
		Status:   "ok",
		Backends: make(map[string]backendStatus, len(h.backends)),
	}

	healthy := true
	for r := range results {
		resp.Backends[r.name] = r.status
		if r.status.Status != "ok" {
			healthy = false
		}
	}

	if !healthy {
		resp.Status = "unavailable"
		h.logger.Warn("readiness check failed", "backends", resp.Backends)
	}

	w.Header().Set("Content-Type", "application/json")
	if healthy {
		w.WriteHeader(http.StatusOK)
	} else {
		w.WriteHeader(http.StatusServiceUnavailable)
	}
	json.NewEncoder(w).Encode(resp)
}
