// Package chatqueue implements the incoming chat-message queue.
package chatqueue

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/hortbot/hortbot/internal/pkg/dbx"
	"github.com/jackc/pgx/v5"
	"github.com/rs/xid"
)

var ErrLeaseLost = errors.New("chatqueue: lease lost")

const (
	notificationChannel = "hortbot_chat_message_queue"

	CleanupBatchSize = 1000
	DedupeDuration   = 5 * time.Minute
	FailedRetention  = 24 * time.Hour
	PendingRetention = time.Hour
)

type Message struct {
	ID               string
	BroadcasterLogin string
	MessageTimestamp time.Time
	EnqueuedAt       time.Time
	Payload          json.RawMessage
}

type Lease struct {
	Message
	Token string
}

type CleanupResult struct {
	Stale     int64
	Completed int64
	Failed    int64
}

type Queue struct {
	db   *sql.DB
	wake chan struct{}
}

func New(db *sql.DB, workers int) *Queue {
	if db == nil {
		panic("nil db")
	}
	if workers <= 0 {
		panic("bad worker count")
	}
	return &Queue{
		db:   db,
		wake: make(chan struct{}, workers),
	}
}

// Enqueue adds a message if its ID is not already queued or retained
// as a completion tombstone. It reports whether a row was inserted.
func (q *Queue) Enqueue(ctx context.Context, message Message) (bool, error) {
	switch {
	case message.ID == "":
		return false, errors.New("message has empty ID")
	case message.BroadcasterLogin == "":
		return false, fmt.Errorf("message %q has empty broadcaster login", message.ID)
	case message.MessageTimestamp.IsZero():
		return false, fmt.Errorf("message %q has zero timestamp", message.ID)
	case message.EnqueuedAt.IsZero():
		return false, fmt.Errorf("message %q has zero enqueue time", message.ID)
	case !json.Valid(message.Payload):
		return false, fmt.Errorf("message %q has invalid JSON payload", message.ID)
	}

	var inserted bool
	err := dbx.Transact(ctx, q.db,
		func(ctx context.Context, tx *sql.Tx) error {
			_, err := tx.ExecContext(ctx, `
				INSERT INTO chat_message_queue_keys (broadcaster_login)
				VALUES ($1)
				ON CONFLICT (broadcaster_login) DO NOTHING
			`, message.BroadcasterLogin)
			if err != nil {
				return fmt.Errorf("insert queue keys: %w", err)
			}
			return nil
		},
		func(ctx context.Context, tx *sql.Tx) error {
			result, err := tx.ExecContext(ctx, `
				INSERT INTO chat_message_queue (
					message_id,
					broadcaster_login,
					message_timestamp,
					enqueued_at,
					payload
				)
				VALUES ($1, $2, $3, $4, $5::jsonb)
				ON CONFLICT (message_id) DO NOTHING
			`, message.ID, message.BroadcasterLogin, message.MessageTimestamp, message.EnqueuedAt, string(message.Payload))
			if err != nil {
				return fmt.Errorf("insert queued message: %w", err)
			}
			n, err := result.RowsAffected()
			if err != nil {
				return fmt.Errorf("get inserted message count: %w", err)
			}
			inserted = n == 1
			return nil
		},
		func(ctx context.Context, tx *sql.Tx) error {
			if !inserted {
				return nil
			}
			if _, err := tx.ExecContext(ctx, `SELECT pg_notify($1, '')`, notificationChannel); err != nil {
				return fmt.Errorf("notify queue workers: %w", err)
			}
			return nil
		},
	)
	if err != nil {
		return false, err
	}
	return inserted, nil
}

func (q *Queue) Listen(ctx context.Context, connString string) error {
	conn, err := pgx.Connect(ctx, connString)
	if err != nil {
		return fmt.Errorf("connect queue listener: %w", err)
	}
	defer conn.Close(context.WithoutCancel(ctx)) //nolint:errcheck

	if _, err := conn.Exec(ctx, `LISTEN `+notificationChannel); err != nil {
		return fmt.Errorf("listen for queued messages: %w", err)
	}
	q.notifyWorker()

	for {
		if _, err := conn.WaitForNotification(ctx); err != nil {
			return fmt.Errorf("wait for queued message notification: %w", err)
		}
		q.notifyWorker()
	}
}

func (q *Queue) Claim(ctx context.Context, leaseDuration time.Duration) (*Lease, error) {
	if leaseDuration <= 0 {
		panic("bad lease duration")
	}

	token := xid.New().String()
	var lease *Lease

	err := dbx.Transact(ctx, q.db, func(ctx context.Context, tx *sql.Tx) error {
		var message Message
		err := tx.QueryRowContext(ctx, `
			SELECT
				q.message_id,
				q.broadcaster_login,
				q.message_timestamp,
				q.enqueued_at,
				q.payload
			FROM chat_message_queue AS q
			JOIN chat_message_queue_keys AS k USING (broadcaster_login)
			WHERE q.completed_at IS NULL
				AND q.failed_at IS NULL
				AND (q.lease_until IS NULL OR q.lease_until <= NOW())
				AND (k.lease_until IS NULL OR k.lease_until <= NOW())
			ORDER BY q.enqueued_at, q.message_id
			FOR UPDATE OF q, k SKIP LOCKED
			LIMIT 1
		`).Scan(
			&message.ID,
			&message.BroadcasterLogin,
			&message.MessageTimestamp,
			&message.EnqueuedAt,
			&message.Payload,
		)
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("select queued message: %w", err)
		}

		var leaseUntil time.Time
		if err := tx.QueryRowContext(ctx, `
			UPDATE chat_message_queue_keys
			SET
				lease_token = $2,
				lease_until = NOW() + ($3 * INTERVAL '1 microsecond')
			WHERE broadcaster_login = $1
			RETURNING lease_until
		`, message.BroadcasterLogin, token, leaseDuration.Microseconds()).Scan(&leaseUntil); err != nil {
			return fmt.Errorf("lease queue key: %w", err)
		}

		if _, err := tx.ExecContext(ctx, `
			UPDATE chat_message_queue
			SET lease_token = $2, lease_until = $3
			WHERE message_id = $1
		`, message.ID, token, leaseUntil); err != nil {
			return fmt.Errorf("lease queued message: %w", err)
		}

		lease = &Lease{
			Message: message,
			Token:   token,
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	if lease != nil {
		q.notifyWorker()
	}
	return lease, nil
}

func (q *Queue) Complete(ctx context.Context, lease *Lease) error {
	if lease == nil {
		panic("nil lease")
	}

	err := dbx.Transact(ctx, q.db, func(ctx context.Context, tx *sql.Tx) error {
		result, err := tx.ExecContext(ctx, `
			UPDATE chat_message_queue
			SET
				completed_at = NOW(),
				lease_token = NULL,
				lease_until = NULL
			WHERE message_id = $1
				AND lease_token = $2
				AND completed_at IS NULL
				AND failed_at IS NULL
		`, lease.ID, lease.Token)
		if err != nil {
			return fmt.Errorf("complete queued message: %w", err)
		}
		n, err := result.RowsAffected()
		if err != nil {
			return fmt.Errorf("get completed message count: %w", err)
		}
		if n != 1 {
			return ErrLeaseLost
		}

		if _, err := tx.ExecContext(ctx, `
			UPDATE chat_message_queue_keys
			SET lease_token = NULL, lease_until = NULL
			WHERE broadcaster_login = $1 AND lease_token = $2
		`, lease.BroadcasterLogin, lease.Token); err != nil {
			return fmt.Errorf("release queue key: %w", err)
		}
		return nil
	})
	if err != nil {
		return err
	}
	return nil
}

func (q *Queue) Fail(ctx context.Context, lease *Lease, cause error) error {
	if lease == nil {
		panic("nil lease")
	}
	if cause == nil {
		panic("nil cause")
	}

	err := dbx.Transact(ctx, q.db, func(ctx context.Context, tx *sql.Tx) error {
		result, err := tx.ExecContext(ctx, `
			UPDATE chat_message_queue
			SET
				failed_at = NOW(),
				last_error = $3,
				lease_token = NULL,
				lease_until = NULL
			WHERE message_id = $1 AND lease_token = $2
		`, lease.ID, lease.Token, cause.Error())
		if err != nil {
			return fmt.Errorf("fail queued message: %w", err)
		}
		n, err := result.RowsAffected()
		if err != nil {
			return fmt.Errorf("get failed message count: %w", err)
		}
		if n != 1 {
			return ErrLeaseLost
		}

		if _, err := tx.ExecContext(ctx, `
			UPDATE chat_message_queue_keys
			SET lease_token = NULL, lease_until = NULL
			WHERE broadcaster_login = $1 AND lease_token = $2
		`, lease.BroadcasterLogin, lease.Token); err != nil {
			return fmt.Errorf("release failed queue key: %w", err)
		}
		return nil
	})
	if err != nil {
		return err
	}
	return nil
}

func (q *Queue) Cleanup(ctx context.Context, staleCutoff, completedCutoff, failedCutoff time.Time, limit int) (CleanupResult, error) {
	if limit <= 0 {
		panic("bad cleanup limit")
	}

	var result CleanupResult
	err := dbx.Transact(ctx, q.db,
		func(ctx context.Context, tx *sql.Tx) error {
			dbResult, err := tx.ExecContext(ctx, `
				WITH doomed AS (
					SELECT message_id
					FROM chat_message_queue
					WHERE message_timestamp <= $1
						AND completed_at IS NULL
						AND failed_at IS NULL
						AND (lease_until IS NULL OR lease_until <= NOW())
					ORDER BY message_timestamp, message_id
					FOR UPDATE SKIP LOCKED
					LIMIT $2
				)
				DELETE FROM chat_message_queue AS q
				USING doomed
				WHERE q.message_id = doomed.message_id
			`, staleCutoff, limit)
			if err != nil {
				return fmt.Errorf("delete stale queued messages: %w", err)
			}
			result.Stale, err = dbResult.RowsAffected()
			if err != nil {
				return fmt.Errorf("get stale message count: %w", err)
			}
			return nil
		},
		func(ctx context.Context, tx *sql.Tx) error {
			dbResult, err := tx.ExecContext(ctx, `
				WITH doomed AS (
					SELECT message_id
					FROM chat_message_queue
					WHERE completed_at <= $1
					ORDER BY completed_at, message_id
					FOR UPDATE SKIP LOCKED
					LIMIT $2
				)
				DELETE FROM chat_message_queue AS q
				USING doomed
				WHERE q.message_id = doomed.message_id
			`, completedCutoff, limit)
			if err != nil {
				return fmt.Errorf("delete completed message tombstones: %w", err)
			}
			result.Completed, err = dbResult.RowsAffected()
			if err != nil {
				return fmt.Errorf("get completed tombstone count: %w", err)
			}
			return nil
		},
		func(ctx context.Context, tx *sql.Tx) error {
			dbResult, err := tx.ExecContext(ctx, `
				WITH doomed AS (
					SELECT message_id
					FROM chat_message_queue
					WHERE failed_at <= $1
					ORDER BY failed_at, message_id
					FOR UPDATE SKIP LOCKED
					LIMIT $2
				)
				DELETE FROM chat_message_queue AS q
				USING doomed
				WHERE q.message_id = doomed.message_id
			`, failedCutoff, limit)
			if err != nil {
				return fmt.Errorf("delete failed messages: %w", err)
			}
			result.Failed, err = dbResult.RowsAffected()
			if err != nil {
				return fmt.Errorf("get failed message count: %w", err)
			}
			return nil
		},
	)
	return result, err
}

func (q *Queue) Wake() <-chan struct{} {
	return q.wake
}

func (q *Queue) notifyWorker() {
	select {
	case q.wake <- struct{}{}:
	default:
	}
}
