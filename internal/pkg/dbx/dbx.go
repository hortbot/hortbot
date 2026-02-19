// Package dbx provides helpers for the database/sql package.
package dbx

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"time"
)

// SetLocalLockTimeout returns a transaction option which will set the lock
// timeout for the transaction.
func SetLocalLockTimeout(timeout time.Duration) func(context.Context, *sql.Tx) error {
	if timeout < 0 {
		panic("duration must be positive")
	}

	ms := timeout.Milliseconds()
	// Postgres refuses to allow "$1" in the SET statement, so construct this as a string.
	//nolint:gosec
	query := "SET LOCAL lock_timeout = " + strconv.FormatInt(ms, 10)

	return func(ctx context.Context, tx *sql.Tx) error {
		if _, err := tx.ExecContext(ctx, query); err != nil {
			return fmt.Errorf("set lock timeout: %w", err)
		}
		return nil
	}
}

// Transact begins a transaction, executes a sequence of functions on that
// transaction, and commits. If any of the functions returns a non-nil error
// or panics, execution is halted and the transaction will be rolled back.
func Transact(ctx context.Context, db *sql.DB, fns ...func(context.Context, *sql.Tx) error) (retErr error) {
	if len(fns) == 0 {
		panic("no fns")
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}

	rollback := true

	defer func() {
		if rollback {
			if err := tx.Rollback(); retErr == nil && err != nil {
				retErr = fmt.Errorf("rollback: %w", err)
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
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit: %w", err)
	}
	return nil
}
