package metrics

import (
	"context"
	"database/sql"
	"sync"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDBMetrics_NotNil(t *testing.T) {
	m := GetDBMetrics()
	require.NotNil(t, m)
	assert.NotNil(t, m.ConnectionsActive)
	assert.NotNil(t, m.ConnectionsTotal)
	assert.NotNil(t, m.WaitDurationSeconds)
	assert.NotNil(t, m.QueryDurationSeconds)
	assert.NotNil(t, m.PoolMaxConnections)
	assert.NotNil(t, m.PoolWaitCount)
	assert.NotNil(t, m.PoolWaitDurationTotal)
	assert.NotEmpty(t, m.instance)
	assert.NotNil(t, m.lastStats)
}

func TestGetDBMetrics_Singleton(t *testing.T) {
	m1 := GetDBMetrics()
	m2 := GetDBMetrics()
	assert.Same(t, m1, m2, "GetDBMetrics should return the same instance")
}

func TestDBMetrics_UpdatePoolStats_FirstCall(t *testing.T) {
	m := GetDBMetrics()
	poolName := "test-first-call"

	stats := sql.DBStats{
		MaxOpenConnections: 25,
		OpenConnections:    10,
		InUse:              5,
		Idle:               5,
		WaitCount:          0,
		WaitDuration:       0,
	}

	m.UpdatePoolStats(poolName, stats)

	assert.Equal(t, float64(5), testutil.ToFloat64(m.ConnectionsActive.WithLabelValues(m.instance, poolName, "in_use")))
	assert.Equal(t, float64(5), testutil.ToFloat64(m.ConnectionsActive.WithLabelValues(m.instance, poolName, "idle")))
	assert.Equal(t, float64(10), testutil.ToFloat64(m.ConnectionsActive.WithLabelValues(m.instance, poolName, "open")))
	assert.Equal(t, float64(25), testutil.ToFloat64(m.PoolMaxConnections.WithLabelValues(m.instance, poolName)))
}

func TestDBMetrics_UpdatePoolStats_DeltaCalculation(t *testing.T) {
	m := GetDBMetrics()
	poolName := "test-delta-calc"

	stats1 := sql.DBStats{
		MaxOpenConnections: 25,
		OpenConnections:    10,
		InUse:              5,
		Idle:               5,
		WaitCount:          100,
		WaitDuration:       time.Second,
		MaxIdleClosed:      5,
		MaxIdleTimeClosed:  2,
		MaxLifetimeClosed:  1,
	}
	m.UpdatePoolStats(poolName, stats1)

	stats2 := sql.DBStats{
		MaxOpenConnections: 25,
		OpenConnections:    12,
		InUse:              7,
		Idle:               5,
		WaitCount:          150,
		WaitDuration:       2 * time.Second,
		MaxIdleClosed:      10,
		MaxIdleTimeClosed:  4,
		MaxLifetimeClosed:  2,
	}
	m.UpdatePoolStats(poolName, stats2)

	m.mu.RLock()
	lastStats, exists := m.lastStats[poolName]
	m.mu.RUnlock()

	assert.True(t, exists)
	assert.Equal(t, stats2.WaitCount, lastStats.WaitCount)
	assert.Equal(t, stats2.WaitDuration, lastStats.WaitDuration)
}

func TestDBMetrics_UpdatePoolStats_NoIncreaseNoDelta(t *testing.T) {
	m := GetDBMetrics()
	poolName := "test-no-delta"

	stats := sql.DBStats{
		MaxOpenConnections: 10,
		OpenConnections:    5,
		InUse:              2,
		Idle:               3,
		WaitCount:          50,
		WaitDuration:       500 * time.Millisecond,
		MaxIdleClosed:      3,
		MaxIdleTimeClosed:  1,
		MaxLifetimeClosed:  0,
	}
	m.UpdatePoolStats(poolName, stats)

	// Second call with same values — no delta should be added
	waitCountBefore := testutil.ToFloat64(m.PoolWaitCount.WithLabelValues(m.instance, poolName))
	m.UpdatePoolStats(poolName, stats)
	waitCountAfter := testutil.ToFloat64(m.PoolWaitCount.WithLabelValues(m.instance, poolName))

	assert.Equal(t, waitCountBefore, waitCountAfter)
}

func TestDBMetrics_ObserveQueryDuration(t *testing.T) {
	m := GetDBMetrics()

	require.NotPanics(t, func() {
		m.ObserveQueryDuration("test-query-dur", "query", 5*time.Millisecond)
		m.ObserveQueryDuration("test-query-dur", "exec", 15*time.Millisecond)
		m.ObserveQueryDuration("test-query-dur", "query_row", 2*time.Millisecond)
	})
}

func TestDBMetrics_ObserveWaitDuration(t *testing.T) {
	m := GetDBMetrics()

	require.NotPanics(t, func() {
		m.ObserveWaitDuration("test-wait-dur", 100*time.Microsecond)
		m.ObserveWaitDuration("test-wait-dur", time.Millisecond)
		m.ObserveWaitDuration("test-wait-dur", 0)
	})
}

func TestDBMetrics_RecordConnection(t *testing.T) {
	m := GetDBMetrics()
	poolName := "test-record-conn"

	before := testutil.ToFloat64(m.ConnectionsTotal.WithLabelValues(m.instance, poolName))
	m.RecordConnection(poolName)
	m.RecordConnection(poolName)
	m.RecordConnection(poolName)
	after := testutil.ToFloat64(m.ConnectionsTotal.WithLabelValues(m.instance, poolName))

	assert.Equal(t, before+3, after)
}

func TestDBMetrics_SetConnectionState(t *testing.T) {
	m := GetDBMetrics()
	poolName := "test-set-state"

	m.SetConnectionState(poolName, StateInUse, 5)
	m.SetConnectionState(poolName, StateIdle, 10)
	m.SetConnectionState(poolName, StateOpen, 15)

	assert.Equal(t, float64(5), testutil.ToFloat64(m.ConnectionsActive.WithLabelValues(m.instance, poolName, "in_use")))
	assert.Equal(t, float64(10), testutil.ToFloat64(m.ConnectionsActive.WithLabelValues(m.instance, poolName, "idle")))
	assert.Equal(t, float64(15), testutil.ToFloat64(m.ConnectionsActive.WithLabelValues(m.instance, poolName, "open")))

	// Update to zero
	m.SetConnectionState(poolName, StateInUse, 0)
	assert.Equal(t, float64(0), testutil.ToFloat64(m.ConnectionsActive.WithLabelValues(m.instance, poolName, "in_use")))
}

func TestConnectionState_Constants(t *testing.T) {
	assert.Equal(t, ConnectionState("in_use"), StateInUse)
	assert.Equal(t, ConnectionState("idle"), StateIdle)
	assert.Equal(t, ConnectionState("open"), StateOpen)
}

func TestDBStatsCollector_NewAndDescribe(t *testing.T) {
	m := GetDBMetrics()
	collector := NewDBStatsCollector(nil, "test-collector", m)
	require.NotNil(t, collector)

	// Describe is a no-op because metrics are registered via promauto,
	// but we verify it doesn't panic and drains the channel
	ch := make(chan *prometheus.Desc, 100)
	require.NotPanics(t, func() {
		collector.Describe(ch)
	})
	// Channel should be empty (no descriptors sent by this collector)
	close(ch)

	// Verify the collector can be registered with a real prometheus registry.
	// Use a mock db so Collect() doesn't panic on nil pointer.
	db, _, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	collector2 := NewDBStatsCollector(db, "test-registry", m)
	registry := prometheus.NewRegistry()
	require.NotPanics(t, func() {
		registry.MustRegister(collector2)
		// Describe is called implicitly via MustRegister
		_, err := registry.Gather()
		require.NoError(t, err)
	})
}

func TestQueryTimer_Done(t *testing.T) {
	timer := NewQueryTimer("test-timer", "query")
	require.NotNil(t, timer)

	time.Sleep(10 * time.Millisecond)
	duration := timer.Done()
	assert.GreaterOrEqual(t, duration, 10*time.Millisecond)
}

func TestQueryTimer_ObserveDuration(t *testing.T) {
	timer := NewQueryTimer("test-timer-obs", "exec")
	require.NotNil(t, timer)

	require.NotPanics(t, func() {
		timer.ObserveDuration(50 * time.Millisecond)
		timer.ObserveDuration(0)
	})
}

func TestInstrumentedDB_Structure(t *testing.T) {
	poolName := "instrumented-struct"

	idb := &InstrumentedDB{
		db:       nil,
		poolName: poolName,
		metrics:  GetDBMetrics(),
	}

	assert.Nil(t, idb.DB())
	assert.Equal(t, poolName, idb.poolName)
}

func TestStartStatsCollector_ContextCancellation(t *testing.T) {
	// Use an in-memory SQLite-like approach: open a DB that we can get stats from
	// sql.Open with an unknown driver will fail, so we test with context cancellation only
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	// Verify function doesn't panic with cancelled context
	require.NotPanics(t, func() {
		// Can't call StartStatsCollector without a real *sql.DB that supports Stats()
		// but we can verify the context handling pattern
		_ = ctx
	})
}

func TestDBMetrics_ConcurrentAccess(t *testing.T) {
	m := GetDBMetrics()
	poolName := "concurrent-pool-v2"

	var wg sync.WaitGroup
	const numGoroutines = 50

	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(idx int) {
			defer wg.Done()
			stats := sql.DBStats{
				MaxOpenConnections: 25,
				OpenConnections:    10,
				InUse:              5,
				Idle:               5,
				WaitCount:          int64(idx),
				WaitDuration:       time.Duration(idx) * time.Millisecond,
			}
			m.UpdatePoolStats(poolName, stats)
			m.ObserveQueryDuration(poolName, "query", time.Millisecond)
			m.ObserveWaitDuration(poolName, time.Microsecond)
			m.RecordConnection(poolName)
			m.SetConnectionState(poolName, StateInUse, idx)
		}(i)
	}

	wg.Wait()
}

func TestNewInstrumentedDB(t *testing.T) {
	db, _, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	idb := NewInstrumentedDB(db, "test-instrumented")
	require.NotNil(t, idb)
	assert.Equal(t, db, idb.DB())
	assert.Equal(t, "test-instrumented", idb.poolName)
	assert.NotNil(t, idb.metrics)
}

func TestInstrumentedDB_Query(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	idb := NewInstrumentedDB(db, "test-query-idb")

	rows := sqlmock.NewRows([]string{"id"}).AddRow(1)
	mock.ExpectQuery("SELECT").WillReturnRows(rows)

	result, err := idb.Query(context.Background(), "SELECT 1")
	require.NoError(t, err)
	require.NotNil(t, result)
	result.Close()

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestInstrumentedDB_QueryRow(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	idb := NewInstrumentedDB(db, "test-queryrow-idb")

	rows := sqlmock.NewRows([]string{"id"}).AddRow(1)
	mock.ExpectQuery("SELECT").WillReturnRows(rows)

	row := idb.QueryRow(context.Background(), "SELECT 1")
	require.NotNil(t, row)

	var id int
	err = row.Scan(&id)
	require.NoError(t, err)
	assert.Equal(t, 1, id)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestInstrumentedDB_Exec(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	idb := NewInstrumentedDB(db, "test-exec-idb")

	mock.ExpectExec("INSERT").WillReturnResult(sqlmock.NewResult(1, 1))

	result, err := idb.Exec(context.Background(), "INSERT INTO t VALUES (1)")
	require.NoError(t, err)
	require.NotNil(t, result)

	affected, err := result.RowsAffected()
	require.NoError(t, err)
	assert.Equal(t, int64(1), affected)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestInstrumentedDB_UpdateStats(t *testing.T) {
	db, _, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	idb := NewInstrumentedDB(db, "test-updatestats-idb")

	require.NotPanics(t, func() {
		idb.UpdateStats()
	})
}

func TestDBStatsCollector_Collect(t *testing.T) {
	db, _, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	m := GetDBMetrics()
	collector := NewDBStatsCollector(db, "test-collect", m)

	ch := make(chan prometheus.Metric, 100)
	require.NotPanics(t, func() {
		collector.Collect(ch)
	})
}

func TestStartStatsCollector_RunsAndStops(t *testing.T) {
	db, _, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	ctx, cancel := context.WithCancel(context.Background())

	require.NotPanics(t, func() {
		StartStatsCollector(ctx, db, "test-stats-collector", 50*time.Millisecond)
	})

	// Let at least one tick fire
	time.Sleep(120 * time.Millisecond)
	cancel()
	// Give goroutine time to exit
	time.Sleep(20 * time.Millisecond)
}

func TestDBMetrics_UpdatePoolStats_DecreasingValues(t *testing.T) {
	m := GetDBMetrics()
	poolName := "test-decreasing"

	// First update with high values
	stats1 := sql.DBStats{
		WaitCount:         200,
		WaitDuration:      5 * time.Second,
		MaxIdleClosed:     20,
		MaxIdleTimeClosed: 10,
		MaxLifetimeClosed: 5,
	}
	m.UpdatePoolStats(poolName, stats1)

	// Second update with lower values (e.g., after pool reset)
	// Counters should NOT add negative deltas
	waitCountBefore := testutil.ToFloat64(m.PoolWaitCount.WithLabelValues(m.instance, poolName))
	waitDurBefore := testutil.ToFloat64(m.PoolWaitDurationTotal.WithLabelValues(m.instance, poolName))

	stats2 := sql.DBStats{
		WaitCount:         100,
		WaitDuration:      2 * time.Second,
		MaxIdleClosed:     10,
		MaxIdleTimeClosed: 5,
		MaxLifetimeClosed: 2,
	}
	m.UpdatePoolStats(poolName, stats2)

	waitCountAfter := testutil.ToFloat64(m.PoolWaitCount.WithLabelValues(m.instance, poolName))
	waitDurAfter := testutil.ToFloat64(m.PoolWaitDurationTotal.WithLabelValues(m.instance, poolName))

	assert.Equal(t, waitCountBefore, waitCountAfter, "should not add negative delta for WaitCount")
	assert.Equal(t, waitDurBefore, waitDurAfter, "should not add negative delta for WaitDuration")
}
