package metrics

import (
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus"
)

// PoolCollector implements prometheus.Collector for pgxpool statistics.
// Stats are read on-demand during each Prometheus scrape — no polling goroutine.
type PoolCollector struct {
	pools map[string]*pgxpool.Pool

	acquireCount           *prometheus.Desc
	acquireDuration        *prometheus.Desc
	acquiredConns          *prometheus.Desc
	canceledAcquireCount   *prometheus.Desc
	constructingConns      *prometheus.Desc
	emptyAcquireCount      *prometheus.Desc
	idleConns              *prometheus.Desc
	maxConns               *prometheus.Desc
	maxIdleDestroyCount    *prometheus.Desc
	maxLifetimeDestroyCount *prometheus.Desc
	newConnsCount          *prometheus.Desc
	totalConns             *prometheus.Desc
}

// NewPoolCollector creates a collector that exports pgxpool stats per backend.
func NewPoolCollector(pools map[string]*pgxpool.Pool) *PoolCollector {
	return &PoolCollector{
		pools: pools,
		acquireCount: prometheus.NewDesc(
			"mezzanine_pgxpool_acquire_count",
			"Cumulative count of successful connection acquires.",
			[]string{"backend"}, nil,
		),
		acquireDuration: prometheus.NewDesc(
			"mezzanine_pgxpool_acquire_duration_seconds",
			"Cumulative time spent acquiring connections.",
			[]string{"backend"}, nil,
		),
		acquiredConns: prometheus.NewDesc(
			"mezzanine_pgxpool_acquired_conns",
			"Number of currently acquired connections.",
			[]string{"backend"}, nil,
		),
		canceledAcquireCount: prometheus.NewDesc(
			"mezzanine_pgxpool_canceled_acquire_count",
			"Cumulative count of acquires canceled by context.",
			[]string{"backend"}, nil,
		),
		constructingConns: prometheus.NewDesc(
			"mezzanine_pgxpool_constructing_conns",
			"Number of connections currently being constructed.",
			[]string{"backend"}, nil,
		),
		emptyAcquireCount: prometheus.NewDesc(
			"mezzanine_pgxpool_empty_acquire_count",
			"Cumulative count of acquires from an empty pool.",
			[]string{"backend"}, nil,
		),
		idleConns: prometheus.NewDesc(
			"mezzanine_pgxpool_idle_conns",
			"Number of idle connections in the pool.",
			[]string{"backend"}, nil,
		),
		maxConns: prometheus.NewDesc(
			"mezzanine_pgxpool_max_conns",
			"Maximum number of connections allowed.",
			[]string{"backend"}, nil,
		),
		maxIdleDestroyCount: prometheus.NewDesc(
			"mezzanine_pgxpool_max_idle_destroy_count",
			"Cumulative count of connections destroyed due to idle timeout.",
			[]string{"backend"}, nil,
		),
		maxLifetimeDestroyCount: prometheus.NewDesc(
			"mezzanine_pgxpool_max_lifetime_destroy_count",
			"Cumulative count of connections destroyed due to max lifetime.",
			[]string{"backend"}, nil,
		),
		newConnsCount: prometheus.NewDesc(
			"mezzanine_pgxpool_new_conns_count",
			"Cumulative count of new connections created.",
			[]string{"backend"}, nil,
		),
		totalConns: prometheus.NewDesc(
			"mezzanine_pgxpool_total_conns",
			"Total number of connections in the pool.",
			[]string{"backend"}, nil,
		),
	}
}

// Describe implements prometheus.Collector.
func (c *PoolCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.acquireCount
	ch <- c.acquireDuration
	ch <- c.acquiredConns
	ch <- c.canceledAcquireCount
	ch <- c.constructingConns
	ch <- c.emptyAcquireCount
	ch <- c.idleConns
	ch <- c.maxConns
	ch <- c.maxIdleDestroyCount
	ch <- c.maxLifetimeDestroyCount
	ch <- c.newConnsCount
	ch <- c.totalConns
}

// Collect implements prometheus.Collector.
func (c *PoolCollector) Collect(ch chan<- prometheus.Metric) {
	for name, pool := range c.pools {
		stat := pool.Stat()

		ch <- prometheus.MustNewConstMetric(c.acquireCount, prometheus.GaugeValue, float64(stat.AcquireCount()), name)
		ch <- prometheus.MustNewConstMetric(c.acquireDuration, prometheus.GaugeValue, stat.AcquireDuration().Seconds(), name)
		ch <- prometheus.MustNewConstMetric(c.acquiredConns, prometheus.GaugeValue, float64(stat.AcquiredConns()), name)
		ch <- prometheus.MustNewConstMetric(c.canceledAcquireCount, prometheus.GaugeValue, float64(stat.CanceledAcquireCount()), name)
		ch <- prometheus.MustNewConstMetric(c.constructingConns, prometheus.GaugeValue, float64(stat.ConstructingConns()), name)
		ch <- prometheus.MustNewConstMetric(c.emptyAcquireCount, prometheus.GaugeValue, float64(stat.EmptyAcquireCount()), name)
		ch <- prometheus.MustNewConstMetric(c.idleConns, prometheus.GaugeValue, float64(stat.IdleConns()), name)
		ch <- prometheus.MustNewConstMetric(c.maxConns, prometheus.GaugeValue, float64(stat.MaxConns()), name)
		ch <- prometheus.MustNewConstMetric(c.maxIdleDestroyCount, prometheus.GaugeValue, float64(stat.MaxIdleDestroyCount()), name)
		ch <- prometheus.MustNewConstMetric(c.maxLifetimeDestroyCount, prometheus.GaugeValue, float64(stat.MaxLifetimeDestroyCount()), name)
		ch <- prometheus.MustNewConstMetric(c.newConnsCount, prometheus.GaugeValue, float64(stat.NewConnsCount()), name)
		ch <- prometheus.MustNewConstMetric(c.totalConns, prometheus.GaugeValue, float64(stat.TotalConns()), name)
	}
}
