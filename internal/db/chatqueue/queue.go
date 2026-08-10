// Package chatqueue implements the incoming chat-message queue.
package chatqueue

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/hortbot/hortbot/internal/db/dbsql"
	"github.com/hortbot/hortbot/internal/pkg/dbx"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
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
	db   *pgxpool.Pool
	wake chan struct{}
}

func New(db *pgxpool.Pool, workers int) *Queue {
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
		func(ctx context.Context, tx pgx.Tx) error {
			err := dbsql.New(tx).ChatQueueEnsureKey(ctx, message.BroadcasterLogin)
			if err != nil {
				return fmt.Errorf("insert queue keys: %w", err)
			}
			return nil
		},
		func(ctx context.Context, tx pgx.Tx) error {
			n, err := dbsql.New(tx).ChatQueueEnqueue(ctx, dbsql.ChatQueueEnqueueParams{
				MessageID:        message.ID,
				BroadcasterLogin: message.BroadcasterLogin,
				MessageTimestamp: dbsql.TimestamptzFrom(message.MessageTimestamp),
				EnqueuedAt:       dbsql.TimestamptzFrom(message.EnqueuedAt),
				Payload:          message.Payload,
			})
			if err != nil {
				return fmt.Errorf("insert queued message: %w", err)
			}
			inserted = n == 1
			return nil
		},
		func(ctx context.Context, tx pgx.Tx) error {
			if !inserted {
				return nil
			}
			if err := dbsql.New(tx).ChatQueueNotify(ctx, notificationChannel); err != nil {
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

	err := dbx.Transact(ctx, q.db, func(ctx context.Context, tx pgx.Tx) error {
		qtx := dbsql.New(tx)
		row, err := qtx.ChatQueueClaim(ctx)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("select queued message: %w", err)
		}
		message := Message{
			ID:               row.MessageID,
			BroadcasterLogin: row.BroadcasterLogin,
			MessageTimestamp: row.MessageTimestamp.Time,
			EnqueuedAt:       row.EnqueuedAt.Time,
			Payload:          row.Payload,
		}

		leaseUntil, err := qtx.ChatQueueLeaseKey(ctx, dbsql.ChatQueueLeaseKeyParams{
			LeaseToken:        token,
			LeaseMicroseconds: leaseDuration.Microseconds(),
			BroadcasterLogin:  message.BroadcasterLogin,
		})
		if err != nil {
			return fmt.Errorf("lease queue key: %w", err)
		}

		if err := qtx.ChatQueueLeaseMessage(ctx, dbsql.ChatQueueLeaseMessageParams{
			LeaseToken: token,
			LeaseUntil: leaseUntil,
			MessageID:  message.ID,
		}); err != nil {
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

	err := dbx.Transact(ctx, q.db, func(ctx context.Context, tx pgx.Tx) error {
		qtx := dbsql.New(tx)
		n, err := qtx.ChatQueueComplete(ctx, dbsql.ChatQueueCompleteParams{
			MessageID:  lease.ID,
			LeaseToken: lease.Token,
		})
		if err != nil {
			return fmt.Errorf("complete queued message: %w", err)
		}
		if n != 1 {
			return ErrLeaseLost
		}

		if err := qtx.ChatQueueReleaseKey(ctx, dbsql.ChatQueueReleaseKeyParams{
			BroadcasterLogin: lease.BroadcasterLogin,
			LeaseToken:       lease.Token,
		}); err != nil {
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

	err := dbx.Transact(ctx, q.db, func(ctx context.Context, tx pgx.Tx) error {
		qtx := dbsql.New(tx)
		n, err := qtx.ChatQueueFail(ctx, dbsql.ChatQueueFailParams{
			LastError:  cause.Error(),
			MessageID:  lease.ID,
			LeaseToken: lease.Token,
		})
		if err != nil {
			return fmt.Errorf("fail queued message: %w", err)
		}
		if n != 1 {
			return ErrLeaseLost
		}

		if err := qtx.ChatQueueReleaseKey(ctx, dbsql.ChatQueueReleaseKeyParams{
			BroadcasterLogin: lease.BroadcasterLogin,
			LeaseToken:       lease.Token,
		}); err != nil {
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
		func(ctx context.Context, tx pgx.Tx) error {
			n, err := dbsql.New(tx).ChatQueueDeleteStale(ctx, dbsql.ChatQueueDeleteStaleParams{
				Cutoff:     dbsql.TimestamptzFrom(staleCutoff),
				BatchLimit: int64(limit),
			})
			if err != nil {
				return fmt.Errorf("delete stale queued messages: %w", err)
			}
			result.Stale = n
			return nil
		},
		func(ctx context.Context, tx pgx.Tx) error {
			n, err := dbsql.New(tx).ChatQueueDeleteCompleted(ctx, dbsql.ChatQueueDeleteCompletedParams{
				Cutoff:     dbsql.TimestamptzFrom(completedCutoff),
				BatchLimit: int64(limit),
			})
			if err != nil {
				return fmt.Errorf("delete completed message tombstones: %w", err)
			}
			result.Completed = n
			return nil
		},
		func(ctx context.Context, tx pgx.Tx) error {
			n, err := dbsql.New(tx).ChatQueueDeleteFailed(ctx, dbsql.ChatQueueDeleteFailedParams{
				Cutoff:     dbsql.TimestamptzFrom(failedCutoff),
				BatchLimit: int64(limit),
			})
			if err != nil {
				return fmt.Errorf("delete failed messages: %w", err)
			}
			result.Failed = n
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
