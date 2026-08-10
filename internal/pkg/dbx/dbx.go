// Package dbx provides helpers for pgx transactions.
package dbx

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// SetLocalLockTimeout returns a transaction option which will set the lock
// timeout for the transaction.
func SetLocalLockTimeout(timeout time.Duration) func(context.Context, pgx.Tx) error {
	if timeout < 0 {
		panic("duration must be positive")
	}

	ms := timeout.Milliseconds()
	// Postgres refuses to allow "$1" in the SET statement, so construct this as a string.
	//nolint:gosec
	query := "SET LOCAL lock_timeout = " + strconv.FormatInt(ms, 10)

	return func(ctx context.Context, tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, query); err != nil {
			return fmt.Errorf("set lock timeout: %w", err)
		}
		return nil
	}
}

// Transact begins a transaction, executes a sequence of functions on that
// transaction, and commits. If any of the functions returns a non-nil error
// or panics, execution is halted and the transaction will be rolled back.
func Transact(ctx context.Context, db *pgxpool.Pool, fns ...func(context.Context, pgx.Tx) error) (retErr error) {
	if len(fns) == 0 {
		panic("no fns")
	}

	tx, err := db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}

	rollback := true

	defer func() {
		if rollback {
			if err := tx.Rollback(context.WithoutCancel(ctx)); retErr == nil && err != nil {
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
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit: %w", err)
	}
	return nil
}
