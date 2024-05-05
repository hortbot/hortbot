// Package dbx provides helpers for the database/sql package.
package dbx

import (
	"context"
	"database/sql"
	"strconv"
	"time"

	"go.opencensus.io/trace"
)

// SetLocalLockTimeout returns a transaction option which will set the lock
// timeout for the transaction.
func SetLocalLockTimeout(timeout time.Duration) func(context.Context, *sql.Tx) error {
	if timeout < 0 {
		panic("duration must be positive")
	}

	ms := timeout.Milliseconds()
	// Postgres refuses to allow "$1" in the SET statement, so construct this as a string.
	query := "SET LOCAL lock_timeout = " + strconv.FormatInt(ms, 10)

	return func(ctx context.Context, tx *sql.Tx) error {
		_, err := tx.ExecContext(ctx, query)
		return err
	}
}

// Transact begins a transaction, executes a sequence of functions on that
// transaction, and commits. If any of the functions returns a non-nil error
// or panics, execution is halted and the transaction will be rolled back.
func Transact(ctx context.Context, db *sql.DB, fns ...func(context.Context, *sql.Tx) error) (retErr error) {
	if len(fns) == 0 {
		panic("no fns")
	}

	ctx, span := trace.StartSpan(ctx, "dbx.Transact")
	defer span.End()

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	rollback := true

	defer func() {
		if rollback {
			if err := tx.Rollback(); retErr == nil && err != nil {
				retErr = err
			}
		}
	}()

	for _, fn := range fns {
		if err := fn(ctx, tx); err != nil {
			return err
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}
	}

	rollback = false
	return tx.Commit()
}
