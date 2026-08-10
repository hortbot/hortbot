package bot

import (
	"context"
	"testing"
	"time"

	"github.com/hortbot/hortbot/internal/db/chatqueue"
	"github.com/hortbot/hortbot/internal/db/migrations"
	"github.com/hortbot/hortbot/internal/pkg/testpostgres"
	"gotest.tools/v3/assert"
)

func TestRunQueueListenerReconnects(t *testing.T) {
	t.Parallel()

	pdb, err := testpostgres.New()
	assert.NilError(t, err)
	t.Cleanup(pdb.Cleanup)
	assert.NilError(t, migrations.Up(pdb.ConnStr(), t.Logf))

	db, err := pdb.Open(t.Context())
	assert.NilError(t, err)
	t.Cleanup(func() {
		db.Close()
	})

	listener := chatqueue.New(db, 1)
	producer := chatqueue.New(db, 1)
	ctx, cancel := context.WithCancel(t.Context())
	errs := make(chan error, 1)
	go func() {
		errs <- runQueueListener(ctx, listener, pdb.ConnStr())
	}()

	waitForWake(t, listener.Wake(), "listener did not become ready")
	assert.NilError(t, pdb.Stop())

	assert.NilError(t, pdb.Start())

	waitForWake(t, listener.Wake(), "listener did not reconnect")
	drainWake(listener.Wake())

	inserted, err := producer.Enqueue(t.Context(), messageForListenerTest("after-reconnect"))
	assert.NilError(t, err)
	assert.Assert(t, inserted)
	waitForWake(t, listener.Wake(), "listener did not receive notification after reconnect")

	cancel()
	assert.ErrorIs(t, <-errs, context.Canceled)
}

func TestCompleteHandledMessageIgnoresWorkerCancellation(t *testing.T) {
	t.Parallel()

	pdb, err := testpostgres.New()
	assert.NilError(t, err)
	t.Cleanup(pdb.Cleanup)
	assert.NilError(t, migrations.Up(pdb.ConnStr(), t.Logf))

	db, err := pdb.Open(t.Context())
	assert.NilError(t, err)
	t.Cleanup(func() {
		db.Close()
	})

	queue := chatqueue.New(db, 1)
	inserted, err := queue.Enqueue(t.Context(), messageForListenerTest("completed-after-cancel"))
	assert.NilError(t, err)
	assert.Assert(t, inserted)

	lease, err := queue.Claim(t.Context(), time.Minute)
	assert.NilError(t, err)

	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	assert.NilError(t, completeHandledMessage(ctx, queue, lease))

	next, err := queue.Claim(t.Context(), time.Minute)
	assert.NilError(t, err)
	assert.Assert(t, next == nil)
}

func waitForWake(t *testing.T, wake <-chan struct{}, message string) {
	t.Helper()
	select {
	case <-wake:
	case <-time.After(15 * time.Second):
		t.Fatal(message)
	}
}

func drainWake(wake <-chan struct{}) {
	for {
		select {
		case <-wake:
		default:
			return
		}
	}
}

func messageForListenerTest(id string) chatqueue.Message {
	now := time.Now()
	return chatqueue.Message{
		ID:               id,
		BroadcasterLogin: "channel",
		MessageTimestamp: now,
		EnqueuedAt:       now,
		Payload:          []byte(`{"message":"` + id + `"}`),
	}
}
