package dbx_test

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/hortbot/hortbot/internal/pkg/assertx"
	"github.com/hortbot/hortbot/internal/pkg/dbx"
	"github.com/hortbot/hortbot/internal/pkg/docker/dpostgres"
	"gotest.tools/v3/assert"
)

func TestSetLocalLockTimeoutBad(t *testing.T) {
	t.Parallel()
	assertx.Panic(t, func() {
		dbx.SetLocalLockTimeout(-1)
	}, "duration must be positive")
}

func TestTransactGood(t *testing.T) {
	t.Parallel()
	db, _, cleanup, err := dpostgres.New()
	assert.NilError(t, err)
	t.Cleanup(cleanup)

	_, err = db.Exec("CREATE TABLE test (id SERIAL PRIMARY KEY, value TEXT)")
	assert.NilError(t, err)

	err = dbx.Transact(context.Background(), db,
		dbx.SetLocalLockTimeout(time.Minute),
		func(ctx context.Context, tx *sql.Tx) error {
			_, err := tx.Exec("INSERT INTO test (value) VALUES ('a')")
			assert.NilError(t, err)
			return nil
		},
		func(ctx context.Context, tx *sql.Tx) error {
			_, err := tx.Exec("INSERT INTO test (value) VALUES ('b')")
			assert.NilError(t, err)
			return nil
		},
	)
	assert.NilError(t, err)

	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM test").Scan(&count)
	assert.NilError(t, err)
	assert.Equal(t, 2, count)
}

func TestTransactNoFns(t *testing.T) {
	t.Parallel()
	db, _, cleanup, err := dpostgres.New()
	assert.NilError(t, err)
	t.Cleanup(cleanup)

	_, err = db.Exec("CREATE TABLE test (id SERIAL PRIMARY KEY, value TEXT)")
	assert.NilError(t, err)

	assertx.Panic(t, func() { _ = dbx.Transact(context.Background(), db) }, "no fns")
}

func TestTransactErr(t *testing.T) {
	t.Parallel()
	db, _, cleanup, err := dpostgres.New()
	assert.NilError(t, err)
	t.Cleanup(cleanup)

	_, err = db.Exec("CREATE TABLE test (id SERIAL PRIMARY KEY, value TEXT)")
	assert.NilError(t, err)

	testErr := errors.New("test error")

	err = dbx.Transact(context.Background(), db, func(ctx context.Context, tx *sql.Tx) error {
		_, err := tx.Exec("INSERT INTO test (value) VALUES ('a')")
		assert.NilError(t, err)
		return testErr
	})
	assert.Equal(t, testErr, err)

	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM test").Scan(&count)
	assert.NilError(t, err)
	assert.Equal(t, 0, count)
}

func TestTransactPanic(t *testing.T) {
	t.Parallel()
	db, _, cleanup, err := dpostgres.New()
	assert.NilError(t, err)
	t.Cleanup(cleanup)

	_, err = db.Exec("CREATE TABLE test (id SERIAL PRIMARY KEY, value TEXT)")
	assert.NilError(t, err)

	testErr := errors.New("test error")

	assertx.Panic(t, func() {
		_ = dbx.Transact(context.Background(), db, func(ctx context.Context, tx *sql.Tx) error {
			_, err := tx.Exec("INSERT INTO test (value) VALUES ('a')")
			assert.NilError(t, err)
			panic(testErr)
		})
	}, testErr)

	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM test").Scan(&count)
	assert.NilError(t, err)
	assert.Equal(t, 0, count)
}

func TestTransactErr2(t *testing.T) {
	t.Parallel()
	db, _, cleanup, err := dpostgres.New()
	assert.NilError(t, err)
	t.Cleanup(cleanup)

	_, err = db.Exec("CREATE TABLE test (id SERIAL PRIMARY KEY, value TEXT)")
	assert.NilError(t, err)

	testErr := errors.New("test error")

	err = dbx.Transact(context.Background(), db,
		func(ctx context.Context, tx *sql.Tx) error {
			_, err := tx.Exec("INSERT INTO test (value) VALUES ('a')")
			assert.NilError(t, err)
			return testErr
		},
		func(ctx context.Context, tx *sql.Tx) error {
			_, err := tx.Exec("INSERT INTO test (value) VALUES ('a')")
			assert.NilError(t, err)
			return nil
		},
	)
	assert.Equal(t, testErr, err)

	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM test").Scan(&count)
	assert.NilError(t, err)
	assert.Equal(t, 0, count)
}

func TestTransactErrCancel(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	db, _, cleanup, err := dpostgres.New()
	assert.NilError(t, err)
	t.Cleanup(cleanup)

	_, err = db.Exec("CREATE TABLE test (id SERIAL PRIMARY KEY, value TEXT)")
	assert.NilError(t, err)

	err = dbx.Transact(ctx, db,
		func(ctx context.Context, tx *sql.Tx) error {
			_, err := tx.Exec("INSERT INTO test (value) VALUES ('a')")
			assert.NilError(t, err)
			cancel()
			return nil
		},
		func(ctx context.Context, tx *sql.Tx) error {
			_, err := tx.Exec("INSERT INTO test (value) VALUES ('a')")
			assert.NilError(t, err)
			return nil
		},
	)
	assert.ErrorContains(t, err, "already been")

	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM test").Scan(&count)
	assert.NilError(t, err)
	assert.Equal(t, 0, count)
}

func TestTransactErrCancelStart(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	db, _, cleanup, err := dpostgres.New()
	assert.NilError(t, err)
	t.Cleanup(cleanup)

	_, err = db.Exec("CREATE TABLE test (id SERIAL PRIMARY KEY, value TEXT)")
	assert.NilError(t, err)

	err = dbx.Transact(ctx, db,
		func(ctx context.Context, tx *sql.Tx) error {
			_, err := tx.Exec("INSERT INTO test (value) VALUES ('a')")
			assert.NilError(t, err)
			return nil
		},
		func(ctx context.Context, tx *sql.Tx) error {
			_, err := tx.Exec("INSERT INTO test (value) VALUES ('a')")
			assert.NilError(t, err)
			return nil
		},
	)
	assert.Equal(t, err, context.Canceled)

	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM test").Scan(&count)
	assert.NilError(t, err)
	assert.Equal(t, 0, count)
}
