package chatqueue_test

import (
	"context"
	"encoding/json"
	"errors"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/hortbot/hortbot/internal/db/chatqueue"
	"github.com/hortbot/hortbot/internal/db/migrations"
	"github.com/hortbot/hortbot/internal/pkg/testpostgres"
	"gotest.tools/v3/assert"
)

func TestQueueLifecycleAndKeySerialization(t *testing.T) {
	t.Parallel()

	db := pool.FreshDB(t)
	q := chatqueue.New(db, 2)
	now := time.Now()
	messages := []chatqueue.Message{
		message("first", "a", now),
		message("second", "a", now.Add(time.Millisecond)),
		message("other", "b", now.Add(2*time.Millisecond)),
	}

	enqueue(t, q, messages...)
	for _, message := range messages {
		inserted, err := q.Enqueue(t.Context(), message)
		assert.NilError(t, err)
		assert.Assert(t, !inserted)
	}

	first, err := q.Claim(t.Context(), time.Minute)
	assert.NilError(t, err)
	assert.Equal(t, first.ID, "first")

	other, err := q.Claim(t.Context(), time.Minute)
	assert.NilError(t, err)
	assert.Equal(t, other.ID, "other")

	blocked, err := q.Claim(t.Context(), time.Minute)
	assert.NilError(t, err)
	assert.Assert(t, blocked == nil)

	assert.NilError(t, q.Complete(t.Context(), first))
	second, err := q.Claim(t.Context(), time.Minute)
	assert.NilError(t, err)
	assert.Equal(t, second.ID, "second")

	assert.NilError(t, q.Complete(t.Context(), second))
	assert.NilError(t, q.Complete(t.Context(), other))

	empty, err := q.Claim(t.Context(), time.Minute)
	assert.NilError(t, err)
	assert.Assert(t, empty == nil)
}

func TestQueueLeaseExpiry(t *testing.T) {
	t.Parallel()

	db := pool.FreshDB(t)
	q := chatqueue.New(db, 1)
	enqueue(t, q, message("message", "channel", time.Now()))

	first, err := q.Claim(t.Context(), time.Millisecond)
	assert.NilError(t, err)
	time.Sleep(10 * time.Millisecond)

	second, err := q.Claim(t.Context(), time.Minute)
	assert.NilError(t, err)
	assert.Equal(t, second.ID, first.ID)
	assert.Assert(t, second.Token != first.Token)

	assert.ErrorIs(t, q.Complete(t.Context(), first), chatqueue.ErrLeaseLost)
	assert.NilError(t, q.Complete(t.Context(), second))
}

func TestQueueListen(t *testing.T) {
	t.Parallel()

	pdb, err := testpostgres.New()
	assert.NilError(t, err)
	t.Cleanup(pdb.Cleanup)
	assert.NilError(t, migrations.Up(pdb.ConnStr(), t.Logf))

	db, err := pdb.Open()
	assert.NilError(t, err)
	t.Cleanup(func() {
		assert.NilError(t, db.Close())
	})

	listener := chatqueue.New(db, 3)
	producer := chatqueue.New(db, 1)
	ctx, cancel := context.WithCancel(t.Context())
	errs := make(chan error, 1)
	go func() {
		errs <- listener.Listen(ctx, pdb.ConnStr())
	}()

	select {
	case <-listener.Wake():
	case <-time.After(5 * time.Second):
		t.Fatal("listener did not become ready")
	}
	assertNoWake(t, listener.Wake())

	inserted, err := producer.Enqueue(t.Context(), message("message", "channel", time.Now()))
	assert.NilError(t, err)
	assert.Assert(t, inserted)
	select {
	case <-listener.Wake():
	case <-time.After(5 * time.Second):
		t.Fatal("listener did not receive notification")
	}
	assertNoWake(t, listener.Wake())

	cancel()
	assert.ErrorIs(t, <-errs, context.Canceled)
}

func TestQueueClaimWakesNextWorker(t *testing.T) {
	t.Parallel()

	db := pool.FreshDB(t)
	q := chatqueue.New(db, 2)
	enqueue(t, q, message("message", "channel", time.Now()))

	lease, err := q.Claim(t.Context(), time.Minute)
	assert.NilError(t, err)
	assert.Assert(t, lease != nil)

	select {
	case <-q.Wake():
	case <-time.After(5 * time.Second):
		t.Fatal("successful claim did not wake next worker")
	}
	assertNoWake(t, q.Wake())

	assert.NilError(t, q.Complete(t.Context(), lease))
	assertNoWake(t, q.Wake())
}

func TestQueueConcurrentClaimsSerializeKey(t *testing.T) {
	t.Parallel()

	db := pool.FreshDB(t)
	q := chatqueue.New(db, 2)
	now := time.Now()
	enqueue(t, q,
		message("first", "a", now),
		message("second", "a", now.Add(time.Millisecond)),
		message("other", "b", now.Add(2*time.Millisecond)),
	)

	start := make(chan struct{})
	leases := make(chan *chatqueue.Lease, 2)
	var wg sync.WaitGroup
	for range 2 {
		wg.Go(func() {
			<-start
			lease, err := q.Claim(t.Context(), time.Minute)
			assert.NilError(t, err)
			leases <- lease
		})
	}
	close(start)
	wg.Wait()
	close(leases)

	var ids []string
	for lease := range leases {
		assert.Assert(t, lease != nil)
		ids = append(ids, lease.ID)
	}
	slices.Sort(ids)
	assert.DeepEqual(t, ids, []string{"first", "other"})
}

func assertNoWake(t *testing.T, wake <-chan struct{}) {
	t.Helper()
	select {
	case <-wake:
		t.Fatal("unexpected worker wake")
	default:
	}
}

func TestQueueFailAndCleanup(t *testing.T) {
	t.Parallel()

	db := pool.FreshDB(t)
	q := chatqueue.New(db, 1)
	now := time.Now()
	enqueue(t, q,
		message("failed", "a", now),
		message("stale", "b", now.Add(-time.Hour)),
	)

	lease, err := q.Claim(t.Context(), time.Minute)
	assert.NilError(t, err)
	assert.Equal(t, lease.ID, "stale")
	assert.NilError(t, q.Fail(t.Context(), lease, errors.New("bad payload")))

	deleted, err := q.Cleanup(t.Context(), now.Add(-time.Minute), now.Add(-time.Hour), now.Add(-time.Hour), chatqueue.CleanupBatchSize)
	assert.NilError(t, err)
	assert.Equal(t, deleted.Stale, int64(0))
	assert.Equal(t, deleted.Completed, int64(0))
	assert.Equal(t, deleted.Failed, int64(0))

	next, err := q.Claim(t.Context(), time.Minute)
	assert.NilError(t, err)
	assert.Equal(t, next.ID, "failed")
	assert.NilError(t, q.Complete(t.Context(), next))

	var failedAt time.Time
	var lastError string
	err = db.QueryRowContext(t.Context(), `
		SELECT failed_at, last_error
		FROM chat_message_queue
		WHERE message_id = 'stale'
	`).Scan(&failedAt, &lastError)
	assert.NilError(t, err)
	assert.Equal(t, lastError, "bad payload")

	deleted, err = q.Cleanup(t.Context(), now.Add(-time.Minute), now.Add(-time.Hour), time.Now().Add(time.Minute), chatqueue.CleanupBatchSize)
	assert.NilError(t, err)
	assert.Equal(t, deleted.Failed, int64(1))
}

func TestQueueCleanupAndCompletedDedupe(t *testing.T) {
	t.Parallel()

	db := pool.FreshDB(t)
	q := chatqueue.New(db, 1)
	now := time.Now()
	stale := message("stale", "a", now.Add(-time.Hour))
	fresh := message("fresh", "b", now)
	enqueue(t, q,
		stale,
		fresh,
	)

	deleted, err := q.Cleanup(t.Context(), now.Add(-time.Minute), now.Add(-time.Hour), now.Add(-time.Hour), chatqueue.CleanupBatchSize)
	assert.NilError(t, err)
	assert.Equal(t, deleted.Stale, int64(1))
	assert.Equal(t, deleted.Completed, int64(0))

	lease, err := q.Claim(t.Context(), time.Minute)
	assert.NilError(t, err)
	assert.Equal(t, lease.ID, "fresh")
	assert.NilError(t, q.Complete(t.Context(), lease))

	inserted, err := q.Enqueue(t.Context(), fresh)
	assert.NilError(t, err)
	assert.Assert(t, !inserted)
	duplicate, err := q.Claim(t.Context(), time.Minute)
	assert.NilError(t, err)
	assert.Assert(t, duplicate == nil)

	deleted, err = q.Cleanup(t.Context(), now.Add(-time.Minute), time.Now().Add(time.Minute), now.Add(-time.Hour), chatqueue.CleanupBatchSize)
	assert.NilError(t, err)
	assert.Equal(t, deleted.Stale, int64(0))
	assert.Equal(t, deleted.Completed, int64(1))

	enqueue(t, q, fresh)
	requeued, err := q.Claim(t.Context(), time.Minute)
	assert.NilError(t, err)
	assert.Equal(t, requeued.ID, fresh.ID)
}

func TestQueueCleanupIsBatched(t *testing.T) {
	t.Parallel()

	db := pool.FreshDB(t)
	q := chatqueue.New(db, 1)
	now := time.Now()
	enqueue(t, q,
		message("stale-1", "a", now.Add(-time.Hour)),
		message("stale-2", "b", now.Add(-time.Hour)),
	)

	deleted, err := q.Cleanup(
		t.Context(),
		now.Add(-time.Minute),
		now.Add(-time.Hour),
		now.Add(-time.Hour),
		1,
	)
	assert.NilError(t, err)
	assert.Equal(t, deleted.Stale, int64(1))

	lease, err := q.Claim(t.Context(), time.Minute)
	assert.NilError(t, err)
	assert.Assert(t, lease != nil)
}

func enqueue(t *testing.T, q *chatqueue.Queue, messages ...chatqueue.Message) {
	t.Helper()
	for _, message := range messages {
		inserted, err := q.Enqueue(t.Context(), message)
		assert.NilError(t, err)
		assert.Assert(t, inserted)
	}
}

func message(id, broadcaster string, timestamp time.Time) chatqueue.Message {
	return chatqueue.Message{
		ID:               id,
		BroadcasterLogin: broadcaster,
		MessageTimestamp: timestamp,
		EnqueuedAt:       timestamp,
		Payload:          json.RawMessage(`{"message":"` + id + `"}`),
	}
}
