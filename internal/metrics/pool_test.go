package metrics

import (
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus"
)

func TestPoolCollector_Describe_EmitsAllDescriptors(t *testing.T) {
	collector := NewPoolCollector(nil)

	ch := make(chan *prometheus.Desc, 20)
	collector.Describe(ch)
	close(ch)

	count := 0
	for range ch {
		count++
	}
	if count != 12 {
		t.Errorf("descriptor count: got %d, want 12", count)
	}
}

func TestPoolCollector_Collect_EmptyPools(t *testing.T) {
	collector := NewPoolCollector(map[string]*pgxpool.Pool{})

	ch := make(chan prometheus.Metric, 20)
	collector.Collect(ch)
	close(ch)

	count := 0
	for range ch {
		count++
	}
	if count != 0 {
		t.Errorf("metric count with empty pools: got %d, want 0", count)
	}
}

func TestPoolCollector_Collect_NilPools(t *testing.T) {
	collector := NewPoolCollector(nil)

	ch := make(chan prometheus.Metric, 20)
	collector.Collect(ch)
	close(ch)

	count := 0
	for range ch {
		count++
	}
	if count != 0 {
		t.Errorf("metric count with nil pools: got %d, want 0", count)
	}
}
