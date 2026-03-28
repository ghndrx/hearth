// Package metrics provides Prometheus metrics collectors for Hearth.
package metrics

import (
	"context"
	"database/sql"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

const (
	dbSubsystem = "db"
)

// DBMetrics holds all database-related Prometheus metrics
type DBMetrics struct {
	// ConnectionsActive tracks currently active database connections by state
	ConnectionsActive *prometheus.GaugeVec

	// ConnectionsTotal tracks total database connections ever made
	ConnectionsTotal *prometheus.CounterVec

	// WaitDurationSeconds tracks time spent waiting for a connection from the pool
	WaitDurationSeconds *prometheus.HistogramVec

	// QueryDurationSeconds tracks database query execution time
	QueryDurationSeconds *prometheus.HistogramVec

	// PoolMaxConnections tracks the maximum number of connections in the pool
	PoolMaxConnections *prometheus.GaugeVec

	// PoolWaitCount tracks total number of times a connection had to be waited for
	PoolWaitCount *prometheus.CounterVec

	// PoolWaitDurationTotal tracks total time spent waiting for connections
	PoolWaitDurationTotal *prometheus.CounterVec

	// instance is the pod/instance name for labeling
	instance string

	// mu protects concurrent access to internal state
	mu sync.RWMutex

	// lastStats tracks previous stats for delta calculations
	lastStats map[string]sql.DBStats
}

// globalDBMetrics is the singleton instance
var globalDBMetrics *DBMetrics
var dbMetricsOnce sync.Once

// NewDBMetrics creates and registers database metrics
func NewDBMetrics() *DBMetrics {
	instance := GetInstanceLabel()

	m := &DBMetrics{
		instance:  instance,
		lastStats: make(map[string]sql.DBStats),

		ConnectionsActive: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Subsystem: dbSubsystem,
				Name:      "connections_active",
				Help:      "Number of currently active database connections by state",
			},
			[]string{"instance", "pool_name", "state"},
		),

		ConnectionsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: dbSubsystem,
				Name:      "connections_total",
				Help:      "Total number of database connections established",
			},
			[]string{"instance", "pool_name"},
		),

		WaitDurationSeconds: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Subsystem: dbSubsystem,
				Name:      "wait_duration_seconds",
				Help:      "Time spent waiting for a connection from the pool",
				Buckets:   []float64{.0001, .0005, .001, .005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5},
			},
			[]string{"instance", "pool_name"},
		),

		QueryDurationSeconds: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Subsystem: dbSubsystem,
				Name:      "query_duration_seconds",
				Help:      "Database query execution time in seconds",
				Buckets:   []float64{.0001, .0005, .001, .005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10},
			},
			[]string{"instance", "pool_name", "operation"},
		),

		PoolMaxConnections: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Subsystem: dbSubsystem,
				Name:      "pool_max_connections",
				Help:      "Maximum number of connections configured for the pool",
			},
			[]string{"instance", "pool_name"},
		),

		PoolWaitCount: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: dbSubsystem,
				Name:      "pool_wait_count_total",
				Help:      "Total number of times a connection had to be waited for",
			},
			[]string{"instance", "pool_name"},
		),

		PoolWaitDurationTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: dbSubsystem,
				Name:      "pool_wait_duration_total_seconds",
				Help:      "Total time spent waiting for connections in seconds",
			},
			[]string{"instance", "pool_name"},
		),
	}

	globalDBMetrics = m
	return m
}

// GetDBMetrics returns the global database metrics instance
func GetDBMetrics() *DBMetrics {
	dbMetricsOnce.Do(func() {
		if globalDBMetrics == nil {
			NewDBMetrics()
		}
	})
	return globalDBMetrics
}

// UpdatePoolStats updates metrics from sql.DBStats
// This should be called periodically (e.g., every 15 seconds)
func (m *DBMetrics) UpdatePoolStats(poolName string, stats sql.DBStats) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Active connections by state
	m.ConnectionsActive.WithLabelValues(m.instance, poolName, "in_use").Set(float64(stats.InUse))
	m.ConnectionsActive.WithLabelValues(m.instance, poolName, "idle").Set(float64(stats.Idle))
	m.ConnectionsActive.WithLabelValues(m.instance, poolName, "open").Set(float64(stats.OpenConnections))

	// Pool configuration
	m.PoolMaxConnections.WithLabelValues(m.instance, poolName).Set(float64(stats.MaxOpenConnections))

	// Calculate deltas for counters
	if lastStats, exists := m.lastStats[poolName]; exists {
		// New connections since last check
		if stats.WaitCount > lastStats.WaitCount {
			waitDelta := stats.WaitCount - lastStats.WaitCount
			m.PoolWaitCount.WithLabelValues(m.instance, poolName).Add(float64(waitDelta))
		}

		// Wait duration since last check
		if stats.WaitDuration > lastStats.WaitDuration {
			durationDelta := stats.WaitDuration - lastStats.WaitDuration
			m.PoolWaitDurationTotal.WithLabelValues(m.instance, poolName).Add(durationDelta.Seconds())
		}

		// Connection count changes (approximate new connections)
		// MaxIdleClosed + MaxIdleTimeClosed + MaxLifetimeClosed can help track churn
		totalClosed := stats.MaxIdleClosed + stats.MaxIdleTimeClosed + stats.MaxLifetimeClosed
		lastTotalClosed := lastStats.MaxIdleClosed + lastStats.MaxIdleTimeClosed + lastStats.MaxLifetimeClosed
		if totalClosed > lastTotalClosed {
			// Connections were closed and likely replaced
			m.ConnectionsTotal.WithLabelValues(m.instance, poolName).Add(float64(totalClosed - lastTotalClosed))
		}
	}

	m.lastStats[poolName] = stats
}

// ObserveWaitDuration records time spent waiting for a connection
func (m *DBMetrics) ObserveWaitDuration(poolName string, duration time.Duration) {
	m.WaitDurationSeconds.WithLabelValues(m.instance, poolName).Observe(duration.Seconds())
}

// ObserveQueryDuration records a query execution time
func (m *DBMetrics) ObserveQueryDuration(poolName, operation string, duration time.Duration) {
	m.QueryDurationSeconds.WithLabelValues(m.instance, poolName, operation).Observe(duration.Seconds())
}

// RecordConnection records a new connection being established
func (m *DBMetrics) RecordConnection(poolName string) {
	m.ConnectionsTotal.WithLabelValues(m.instance, poolName).Inc()
}

// ConnectionState represents the state of database connections
type ConnectionState string

const (
	StateInUse ConnectionState = "in_use"
	StateIdle  ConnectionState = "idle"
	StateOpen  ConnectionState = "open"
)

// SetConnectionState sets the connection count for a specific state
func (m *DBMetrics) SetConnectionState(poolName string, state ConnectionState, count int) {
	m.ConnectionsActive.WithLabelValues(m.instance, poolName, string(state)).Set(float64(count))
}

// DBStatsCollector implements prometheus.Collector to automatically collect DB stats
type DBStatsCollector struct {
	db       *sql.DB
	poolName string
	metrics  *DBMetrics
}

// NewDBStatsCollector creates a collector that auto-updates pool stats
func NewDBStatsCollector(db *sql.DB, poolName string, metrics *DBMetrics) *DBStatsCollector {
	return &DBStatsCollector{
		db:       db,
		poolName: poolName,
		metrics:  metrics,
	}
}

// Describe implements prometheus.Collector
func (c *DBStatsCollector) Describe(ch chan<- *prometheus.Desc) {
	// Metrics are already registered via promauto, so we don't describe them here
}

// Collect implements prometheus.Collector
func (c *DBStatsCollector) Collect(ch chan<- prometheus.Metric) {
	stats := c.db.Stats()
	c.metrics.UpdatePoolStats(c.poolName, stats)
}

// InstrumentedDB wraps a sql.DB with metrics collection
type InstrumentedDB struct {
	db       *sql.DB
	poolName string
	metrics  *DBMetrics
}

// NewInstrumentedDB creates a new instrumented database wrapper
func NewInstrumentedDB(db *sql.DB, poolName string) *InstrumentedDB {
	return &InstrumentedDB{
		db:       db,
		poolName: poolName,
		metrics:  GetDBMetrics(),
	}
}

// Query executes a query with metrics collection
func (idb *InstrumentedDB) Query(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	start := time.Now()
	rows, err := idb.db.QueryContext(ctx, query, args...)
	idb.metrics.ObserveQueryDuration(idb.poolName, "query", time.Since(start))
	return rows, err
}

// QueryRow executes a query that returns at most one row
func (idb *InstrumentedDB) QueryRow(ctx context.Context, query string, args ...interface{}) *sql.Row {
	start := time.Now()
	row := idb.db.QueryRowContext(ctx, query, args...)
	idb.metrics.ObserveQueryDuration(idb.poolName, "query_row", time.Since(start))
	return row
}

// Exec executes a query without returning rows
func (idb *InstrumentedDB) Exec(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	start := time.Now()
	result, err := idb.db.ExecContext(ctx, query, args...)
	idb.metrics.ObserveQueryDuration(idb.poolName, "exec", time.Since(start))
	return result, err
}

// UpdateStats updates pool statistics
func (idb *InstrumentedDB) UpdateStats() {
	stats := idb.db.Stats()
	idb.metrics.UpdatePoolStats(idb.poolName, stats)
}

// DB returns the underlying sql.DB
func (idb *InstrumentedDB) DB() *sql.DB {
	return idb.db
}

// StartStatsCollector starts a goroutine that periodically collects DB stats
func StartStatsCollector(ctx context.Context, db *sql.DB, poolName string, interval time.Duration) {
	metrics := GetDBMetrics()

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		// Collect initial stats
		metrics.UpdatePoolStats(poolName, db.Stats())

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				metrics.UpdatePoolStats(poolName, db.Stats())
			}
		}
	}()
}

// QueryTimer is a helper for timing queries
type QueryTimer struct {
	start     time.Time
	poolName  string
	operation string
	metrics   *DBMetrics
}

// NewQueryTimer creates a new query timer
func NewQueryTimer(poolName, operation string) *QueryTimer {
	return &QueryTimer{
		start:     time.Now(),
		poolName:  poolName,
		operation: operation,
		metrics:   GetDBMetrics(),
	}
}

// Done records the query duration
func (t *QueryTimer) Done() time.Duration {
	duration := time.Since(t.start)
	t.metrics.ObserveQueryDuration(t.poolName, t.operation, duration)
	return duration
}

// ObserveDuration records a specific duration (for external timing)
func (t *QueryTimer) ObserveDuration(d time.Duration) {
	t.metrics.ObserveQueryDuration(t.poolName, t.operation, d)
}
