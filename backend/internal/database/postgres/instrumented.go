// Package postgres provides PostgreSQL database access with instrumentation.
package postgres

import (
	"context"
	"database/sql"
	"time"

	"github.com/jmoiron/sqlx"

	"hearth/internal/metrics"
)

// InstrumentedSqlxDB wraps a sqlx.DB with metrics collection
type InstrumentedSqlxDB struct {
	*sqlx.DB
	poolName string
	metrics  *metrics.DBMetrics
}

// NewInstrumentedSqlxDB creates a new instrumented sqlx.DB wrapper
func NewInstrumentedSqlxDB(db *sqlx.DB, poolName string) *InstrumentedSqlxDB {
	return &InstrumentedSqlxDB{
		DB:       db,
		poolName: poolName,
		metrics:  metrics.GetDBMetrics(),
	}
}

// WrapDB wraps an existing sqlx.DB with instrumentation
func WrapDB(db *sqlx.DB, poolName string) *InstrumentedSqlxDB {
	return NewInstrumentedSqlxDB(db, poolName)
}

// QueryxContext executes a query with metrics collection
func (idb *InstrumentedSqlxDB) QueryxContext(ctx context.Context, query string, args ...interface{}) (*sqlx.Rows, error) {
	timer := metrics.NewQueryTimer(idb.poolName, "query")
	rows, err := idb.DB.QueryxContext(ctx, query, args...)
	timer.Done()
	return rows, err
}

// QueryRowxContext executes a query that returns at most one row
func (idb *InstrumentedSqlxDB) QueryRowxContext(ctx context.Context, query string, args ...interface{}) *sqlx.Row {
	timer := metrics.NewQueryTimer(idb.poolName, "query_row")
	row := idb.DB.QueryRowxContext(ctx, query, args...)
	timer.Done()
	return row
}

// GetContext queries and scans a single row
func (idb *InstrumentedSqlxDB) GetContext(ctx context.Context, dest interface{}, query string, args ...interface{}) error {
	timer := metrics.NewQueryTimer(idb.poolName, "get")
	err := idb.DB.GetContext(ctx, dest, query, args...)
	timer.Done()
	return err
}

// SelectContext queries and scans multiple rows
func (idb *InstrumentedSqlxDB) SelectContext(ctx context.Context, dest interface{}, query string, args ...interface{}) error {
	timer := metrics.NewQueryTimer(idb.poolName, "select")
	err := idb.DB.SelectContext(ctx, dest, query, args...)
	timer.Done()
	return err
}

// ExecContext executes a query without returning rows
func (idb *InstrumentedSqlxDB) ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	timer := metrics.NewQueryTimer(idb.poolName, "exec")
	result, err := idb.DB.ExecContext(ctx, query, args...)
	timer.Done()
	return result, err
}

// NamedExecContext executes a named query
func (idb *InstrumentedSqlxDB) NamedExecContext(ctx context.Context, query string, arg interface{}) (sql.Result, error) {
	timer := metrics.NewQueryTimer(idb.poolName, "named_exec")
	result, err := idb.DB.NamedExecContext(ctx, query, arg)
	timer.Done()
	return result, err
}

// PrepareNamedContext prepares a named statement
func (idb *InstrumentedSqlxDB) PrepareNamedContext(ctx context.Context, query string) (*sqlx.NamedStmt, error) {
	timer := metrics.NewQueryTimer(idb.poolName, "prepare_named")
	stmt, err := idb.DB.PrepareNamedContext(ctx, query)
	timer.Done()
	return stmt, err
}

// PreparexContext prepares a statement
func (idb *InstrumentedSqlxDB) PreparexContext(ctx context.Context, query string) (*sqlx.Stmt, error) {
	timer := metrics.NewQueryTimer(idb.poolName, "preparex")
	stmt, err := idb.DB.PreparexContext(ctx, query)
	timer.Done()
	return stmt, err
}

// BeginTxx starts a transaction with options
func (idb *InstrumentedSqlxDB) BeginTxx(ctx context.Context, opts *sql.TxOptions) (*InstrumentedTx, error) {
	timer := metrics.NewQueryTimer(idb.poolName, "begin_tx")
	tx, err := idb.DB.BeginTxx(ctx, opts)
	timer.Done()
	if err != nil {
		return nil, err
	}
	return &InstrumentedTx{
		Tx:       tx,
		poolName: idb.poolName,
		metrics:  idb.metrics,
	}, nil
}

// UpdateStats updates pool statistics
func (idb *InstrumentedSqlxDB) UpdateStats() {
	stats := idb.DB.Stats()
	idb.metrics.UpdatePoolStats(idb.poolName, stats)
}

// Stats returns the underlying DB stats
func (idb *InstrumentedSqlxDB) Stats() sql.DBStats {
	return idb.DB.Stats()
}

// PoolName returns the pool name
func (idb *InstrumentedSqlxDB) PoolName() string {
	return idb.poolName
}

// Unwrap returns the underlying sqlx.DB
func (idb *InstrumentedSqlxDB) Unwrap() *sqlx.DB {
	return idb.DB
}

// InstrumentedTx wraps a sqlx.Tx with metrics collection
type InstrumentedTx struct {
	*sqlx.Tx
	poolName string
	metrics  *metrics.DBMetrics
}

// QueryxContext executes a query in the transaction
func (itx *InstrumentedTx) QueryxContext(ctx context.Context, query string, args ...interface{}) (*sqlx.Rows, error) {
	timer := metrics.NewQueryTimer(itx.poolName, "tx_query")
	rows, err := itx.Tx.QueryxContext(ctx, query, args...)
	timer.Done()
	return rows, err
}

// QueryRowxContext executes a query that returns at most one row
func (itx *InstrumentedTx) QueryRowxContext(ctx context.Context, query string, args ...interface{}) *sqlx.Row {
	timer := metrics.NewQueryTimer(itx.poolName, "tx_query_row")
	row := itx.Tx.QueryRowxContext(ctx, query, args...)
	timer.Done()
	return row
}

// GetContext queries and scans a single row in the transaction
func (itx *InstrumentedTx) GetContext(ctx context.Context, dest interface{}, query string, args ...interface{}) error {
	timer := metrics.NewQueryTimer(itx.poolName, "tx_get")
	err := itx.Tx.GetContext(ctx, dest, query, args...)
	timer.Done()
	return err
}

// SelectContext queries and scans multiple rows in the transaction
func (itx *InstrumentedTx) SelectContext(ctx context.Context, dest interface{}, query string, args ...interface{}) error {
	timer := metrics.NewQueryTimer(itx.poolName, "tx_select")
	err := itx.Tx.SelectContext(ctx, dest, query, args...)
	timer.Done()
	return err
}

// ExecContext executes a query in the transaction
func (itx *InstrumentedTx) ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	timer := metrics.NewQueryTimer(itx.poolName, "tx_exec")
	result, err := itx.Tx.ExecContext(ctx, query, args...)
	timer.Done()
	return result, err
}

// NamedExecContext executes a named query in the transaction
func (itx *InstrumentedTx) NamedExecContext(ctx context.Context, query string, arg interface{}) (sql.Result, error) {
	timer := metrics.NewQueryTimer(itx.poolName, "tx_named_exec")
	result, err := itx.Tx.NamedExecContext(ctx, query, arg)
	timer.Done()
	return result, err
}

// Commit commits the transaction
func (itx *InstrumentedTx) Commit() error {
	timer := metrics.NewQueryTimer(itx.poolName, "tx_commit")
	err := itx.Tx.Commit()
	timer.Done()
	return err
}

// Rollback rolls back the transaction
func (itx *InstrumentedTx) Rollback() error {
	timer := metrics.NewQueryTimer(itx.poolName, "tx_rollback")
	err := itx.Tx.Rollback()
	timer.Done()
	return err
}

// Unwrap returns the underlying sqlx.Tx
func (itx *InstrumentedTx) Unwrap() *sqlx.Tx {
	return itx.Tx
}

// StartDBStatsCollector starts a goroutine that periodically collects DB stats
func StartDBStatsCollector(ctx context.Context, db *sqlx.DB, poolName string, interval time.Duration) {
	metrics.StartStatsCollector(ctx, db.DB, poolName, interval)
}

// NewInstrumentedDB creates a new instrumented database connection
// This is a convenience function that creates a new DB and wraps it
func NewInstrumentedDB(cfg Config, poolName string) (*InstrumentedSqlxDB, error) {
	db, err := NewDB(cfg)
	if err != nil {
		return nil, err
	}
	return NewInstrumentedSqlxDB(db, poolName), nil
}

// NewInstrumentedDBFromURL creates an instrumented database connection from URL
func NewInstrumentedDBFromURL(databaseURL, poolName string) (*InstrumentedSqlxDB, error) {
	db, err := NewDBFromURL(databaseURL)
	if err != nil {
		return nil, err
	}
	return NewInstrumentedSqlxDB(db, poolName), nil
}
