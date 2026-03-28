package postgres

import (
	"context"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newMockDB(t *testing.T) (*sqlx.DB, sqlmock.Sqlmock) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	return sqlx.NewDb(db, "postgres"), mock
}

func TestWrapDB(t *testing.T) {
	db, mock := newMockDB(t)
	defer db.Close()

	// Set an expectation to verify mock is used
	mock.ExpectPing()

	wrapped := WrapDB(db, "test-pool")
	assert.NotNil(t, wrapped)
	assert.Equal(t, "test-pool", wrapped.poolName)
	assert.Equal(t, db, wrapped.Unwrap())
}

func TestNewInstrumentedSqlxDB(t *testing.T) {
	db, mock := newMockDB(t)
	defer db.Close()
	_ = mock // db is used via idb methods

	idb := NewInstrumentedSqlxDB(db, "my-pool")
	assert.NotNil(t, idb)
	assert.Equal(t, "my-pool", idb.poolName)
}

func TestInstrumentedSqlxDB_ExecContext(t *testing.T) {
	db, mock := newMockDB(t)
	defer db.Close()

	idb := NewInstrumentedSqlxDB(db, "test-pool")
	ctx := context.Background()

	mock.ExpectExec("INSERT INTO").WillReturnResult(sqlmock.NewResult(1, 1))
	result, err := idb.ExecContext(ctx, "INSERT INTO test VALUES ($1)", "value")
	require.NoError(t, err)
	affected, err := result.RowsAffected()
	require.NoError(t, err)
	assert.Equal(t, int64(1), affected)
}

func TestInstrumentedSqlxDB_GetContext(t *testing.T) {
	db, mock := newMockDB(t)
	defer db.Close()

	idb := NewInstrumentedSqlxDB(db, "test-pool")
	ctx := context.Background()

	rows := sqlmock.NewRows([]string{"id", "name"}).AddRow(1, "test")
	mock.ExpectQuery("SELECT (.+) FROM test").WillReturnRows(rows)

	var result struct {
		ID   int
		Name string
	}
	err := idb.GetContext(ctx, &result, "SELECT id, name FROM test WHERE id = $1", 1)
	require.NoError(t, err)
	assert.Equal(t, 1, result.ID)
	assert.Equal(t, "test", result.Name)
}

func TestInstrumentedSqlxDB_SelectContext(t *testing.T) {
	db, mock := newMockDB(t)
	defer db.Close()

	idb := NewInstrumentedSqlxDB(db, "test-pool")
	ctx := context.Background()

	rows := sqlmock.NewRows([]string{"id", "name"}).
		AddRow(1, "a").
		AddRow(2, "b")
	mock.ExpectQuery("SELECT (.+) FROM test").WillReturnRows(rows)

	var results []struct {
		ID   int
		Name string
	}
	err := idb.SelectContext(ctx, &results, "SELECT id, name FROM test")
	require.NoError(t, err)
	assert.Len(t, results, 2)
}

func TestInstrumentedSqlxDB_QueryxContext(t *testing.T) {
	db, mock := newMockDB(t)
	defer db.Close()

	idb := NewInstrumentedSqlxDB(db, "test-pool")
	ctx := context.Background()

	rows := sqlmock.NewRows([]string{"id", "name"}).
		AddRow(1, "a").
		AddRow(2, "b")
	mock.ExpectQuery("SELECT (.+) FROM test").WillReturnRows(rows)

	resultRows, err := idb.QueryxContext(ctx, "SELECT id, name FROM test")
	require.NoError(t, err)
	defer resultRows.Close()

	count := 0
	for resultRows.Next() {
		count++
	}
	assert.Equal(t, 2, count)
}

func TestInstrumentedSqlxDB_NamedExecContext(t *testing.T) {
	db, mock := newMockDB(t)
	defer db.Close()

	idb := NewInstrumentedSqlxDB(db, "test-pool")
	ctx := context.Background()

	mock.ExpectExec("INSERT INTO test").WillReturnResult(sqlmock.NewResult(1, 1))
	result, err := idb.NamedExecContext(ctx, "INSERT INTO test (name) VALUES (:name)", map[string]interface{}{"name": "test"})
	require.NoError(t, err)
	affected, err := result.RowsAffected()
	require.NoError(t, err)
	assert.Equal(t, int64(1), affected)
}

func TestInstrumentedSqlxDB_PrepareNamedContext(t *testing.T) {
	db, mock := newMockDB(t)
	defer db.Close()

	idb := NewInstrumentedSqlxDB(db, "test-pool")
	ctx := context.Background()

	mock.ExpectPrepare("SELECT (.+) FROM test")
	stmt, err := idb.PrepareNamedContext(ctx, "SELECT id, name FROM test WHERE id = :id")
	require.NoError(t, err)
	assert.NotNil(t, stmt)
}

func TestInstrumentedSqlxDB_PreparexContext(t *testing.T) {
	db, mock := newMockDB(t)
	defer db.Close()

	idb := NewInstrumentedSqlxDB(db, "test-pool")
	ctx := context.Background()

	mock.ExpectPrepare("SELECT (.+) FROM test WHERE id = \\$1")
	stmt, err := idb.PreparexContext(ctx, "SELECT id, name FROM test WHERE id = $1")
	require.NoError(t, err)
	assert.NotNil(t, stmt)
}

func TestInstrumentedSqlxDB_BeginTxx(t *testing.T) {
	db, mock := newMockDB(t)
	defer db.Close()

	idb := NewInstrumentedSqlxDB(db, "test-pool")
	ctx := context.Background()

	mock.ExpectBegin()
	tx, err := idb.BeginTxx(ctx, nil)
	require.NoError(t, err)
	assert.NotNil(t, tx)
}

func TestInstrumentedSqlxDB_Stats(t *testing.T) {
	db, _ := newMockDB(t)
	defer db.Close()

	idb := NewInstrumentedSqlxDB(db, "test-pool")
	stats := idb.Stats()
	assert.NotNil(t, stats)
}

func TestInstrumentedSqlxDB_PoolName(t *testing.T) {
	db, _ := newMockDB(t)
	defer db.Close()

	idb := NewInstrumentedSqlxDB(db, "my-special-pool")
	assert.Equal(t, "my-special-pool", idb.PoolName())
}

func TestInstrumentedTx_Commit(t *testing.T) {
	db, mock := newMockDB(t)
	defer db.Close()

	idb := NewInstrumentedSqlxDB(db, "test-pool")
	ctx := context.Background()

	mock.ExpectBegin()
	mock.ExpectCommit()

	tx, err := idb.BeginTxx(ctx, nil)
	require.NoError(t, err)

	err = tx.Commit()
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestInstrumentedTx_Rollback(t *testing.T) {
	db, mock := newMockDB(t)
	defer db.Close()

	idb := NewInstrumentedSqlxDB(db, "test-pool")
	ctx := context.Background()

	mock.ExpectBegin()
	mock.ExpectRollback()

	tx, err := idb.BeginTxx(ctx, nil)
	require.NoError(t, err)

	err = tx.Rollback()
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestInstrumentedTx_GetContext(t *testing.T) {
	db, mock := newMockDB(t)
	defer db.Close()

	idb := NewInstrumentedSqlxDB(db, "test-pool")
	ctx := context.Background()

	mock.ExpectBegin()
	tx, err := idb.BeginTxx(ctx, nil)
	require.NoError(t, err)

	rows := sqlmock.NewRows([]string{"id", "name"}).AddRow(1, "test")
	mock.ExpectQuery("SELECT (.+) FROM test").WillReturnRows(rows)

	var result struct {
		ID   int
		Name string
	}
	err = tx.GetContext(ctx, &result, "SELECT id, name FROM test WHERE id = $1", 1)
	require.NoError(t, err)
	assert.Equal(t, 1, result.ID)

	tx.Rollback()
}

func TestInstrumentedTx_SelectContext(t *testing.T) {
	db, mock := newMockDB(t)
	defer db.Close()

	idb := NewInstrumentedSqlxDB(db, "test-pool")
	ctx := context.Background()

	mock.ExpectBegin()
	tx, err := idb.BeginTxx(ctx, nil)
	require.NoError(t, err)

	rows := sqlmock.NewRows([]string{"id", "name"}).
		AddRow(1, "a").
		AddRow(2, "b")
	mock.ExpectQuery("SELECT (.+) FROM test").WillReturnRows(rows)

	var results []struct {
		ID   int
		Name string
	}
	err = tx.SelectContext(ctx, &results, "SELECT id, name FROM test")
	require.NoError(t, err)
	assert.Len(t, results, 2)

	tx.Rollback()
}

func TestInstrumentedTx_ExecContext(t *testing.T) {
	db, mock := newMockDB(t)
	defer db.Close()

	idb := NewInstrumentedSqlxDB(db, "test-pool")
	ctx := context.Background()

	mock.ExpectBegin()
	tx, err := idb.BeginTxx(ctx, nil)
	require.NoError(t, err)

	mock.ExpectExec("INSERT INTO").WillReturnResult(sqlmock.NewResult(1, 1))
	result, err := tx.ExecContext(ctx, "INSERT INTO test VALUES ($1)", "value")
	require.NoError(t, err)
	affected, err := result.RowsAffected()
	require.NoError(t, err)
	assert.Equal(t, int64(1), affected)

	tx.Rollback()
}

func TestInstrumentedTx_NamedExecContext(t *testing.T) {
	db, mock := newMockDB(t)
	defer db.Close()

	idb := NewInstrumentedSqlxDB(db, "test-pool")
	ctx := context.Background()

	mock.ExpectBegin()
	tx, err := idb.BeginTxx(ctx, nil)
	require.NoError(t, err)

	mock.ExpectExec("INSERT INTO test").WillReturnResult(sqlmock.NewResult(1, 1))
	result, err := tx.NamedExecContext(ctx, "INSERT INTO test (name) VALUES (:name)", map[string]interface{}{"name": "test"})
	require.NoError(t, err)
	affected, err := result.RowsAffected()
	require.NoError(t, err)
	assert.Equal(t, int64(1), affected)

	tx.Rollback()
}

func TestInstrumentedTx_Unwrap(t *testing.T) {
	db, mock := newMockDB(t)
	defer db.Close()

	idb := NewInstrumentedSqlxDB(db, "test-pool")
	ctx := context.Background()

	mock.ExpectBegin()
	tx, err := idb.BeginTxx(ctx, nil)
	require.NoError(t, err)

	assert.NotNil(t, tx.Unwrap())
	tx.Rollback()
}

func TestNewInstrumentedDB(t *testing.T) {
	cfg := Config{
		Host:     "localhost",
		Port:     5432,
		User:     "test",
		Password: "test",
		Database: "test",
		SSLMode:  "disable",
	}

	idb, err := NewInstrumentedDB(cfg, "test-pool")
	if err != nil {
		t.Skip("Database not available, skipping integration test")
	}
	assert.NotNil(t, idb)
	idb.Close()
}

func TestNewInstrumentedDBFromURL(t *testing.T) {
	url := "postgres://test:test@localhost:5432/test?sslmode=disable"
	idb, err := NewInstrumentedDBFromURL(url, "test-pool")
	if err != nil {
		t.Skip("Database not available, skipping integration test")
	}
	assert.NotNil(t, idb)
	idb.Close()
}

func TestStartDBStatsCollector(t *testing.T) {
	cfg := Config{
		Host:     "localhost",
		Port:     5432,
		User:     "test",
		Password: "test",
		Database: "test",
		SSLMode:  "disable",
	}

	db, err := NewDB(cfg)
	if err != nil {
		t.Skip("Database not available, skipping test")
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	StartDBStatsCollector(ctx, db, "test-pool", time.Millisecond)
	// Test that it doesn't panic
}
