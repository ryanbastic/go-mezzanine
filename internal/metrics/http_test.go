package metrics

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
)

func TestMetrics_Passthrough(t *testing.T) {
	called := false
	handler := Metrics(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/v1/livez", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if !called {
		t.Error("inner handler was not called")
	}
	if w.Code != http.StatusOK {
		t.Errorf("status: got %d, want %d", w.Code, http.StatusOK)
	}
}

func TestMetrics_CapturesStatus(t *testing.T) {
	handler := Metrics(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status: got %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestMetrics_InFlightReturnsToZero(t *testing.T) {
	handler := Metrics(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// While inside, in-flight should be > 0
		val := testutil.ToFloat64(requestsInFlight)
		if val < 1 {
			t.Errorf("in-flight during request: got %f, want >= 1", val)
		}
		w.WriteHeader(http.StatusOK)
	}))

	before := testutil.ToFloat64(requestsInFlight)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	after := testutil.ToFloat64(requestsInFlight)
	if after != before {
		t.Errorf("in-flight after request: got %f, want %f", after, before)
	}
}

func TestMetrics_RecordsRoutePattern(t *testing.T) {
	mux := chi.NewRouter()
	mux.Use(Metrics)
	mux.Get("/v1/cells/{rowKey}", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Snapshot counter before request
	before := counterValue(t, "mezzanine_requests_total", "GET", "/v1/cells/{rowKey}", "200")

	req := httptest.NewRequest(http.MethodGet, "/v1/cells/550e8400-e29b-41d4-a716-446655440000", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	after := counterValue(t, "mezzanine_requests_total", "GET", "/v1/cells/{rowKey}", "200")
	if after-before != 1 {
		t.Errorf("requests_total delta: got %f, want 1", after-before)
	}
}

// counterValue reads the current value of requests_total for the given labels.
func counterValue(t *testing.T, name, method, route, status string) float64 {
	t.Helper()
	metrics, err := prometheus.DefaultGatherer.Gather()
	if err != nil {
		t.Fatalf("gather: %v", err)
	}
	for _, mf := range metrics {
		if mf.GetName() != name {
			continue
		}
		for _, m := range mf.GetMetric() {
			labels := map[string]string{}
			for _, lp := range m.GetLabel() {
				labels[lp.GetName()] = lp.GetValue()
			}
			if labels["method"] == method && labels["route"] == route && labels["status"] == status {
				return m.GetCounter().GetValue()
			}
		}
	}
	return 0
}
