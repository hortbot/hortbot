package conduit

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/hortbot/hortbot/internal/db/chatqueue"
	"github.com/hortbot/hortbot/internal/db/migrations"
	"github.com/hortbot/hortbot/internal/pkg/apiclient/twitch/eventsub"
	"github.com/hortbot/hortbot/internal/pkg/apiclient/twitch/idstr"
	"github.com/hortbot/hortbot/internal/pkg/testpostgres"
	"gotest.tools/v3/assert"
)

func TestQueuedMessageRoundTrip(t *testing.T) {
	t.Parallel()

	timestamp := time.Now()
	message := testWebsocketMessage(timestamp)
	raw, err := json.Marshal(message)
	assert.NilError(t, err)

	queued, err := queuedMessage(raw, message)
	assert.NilError(t, err)
	assert.Equal(t, queued.ID, "notification")
	assert.Equal(t, queued.BroadcasterLogin, "channel")
	assert.Equal(t, queued.MessageTimestamp, timestamp)
	assert.DeepEqual(t, queued.Payload, json.RawMessage(raw))

	var roundTrip eventsub.WebsocketMessage
	assert.NilError(t, json.Unmarshal(queued.Payload, &roundTrip))
	event := roundTrip.Payload.(*eventsub.NotificationPayload).Event.(*eventsub.ChatMessageEvent)
	assert.Equal(t, event.MessageID, "message")
	assert.Equal(t, event.BroadcasterUserLogin, "channel")
}

func TestNotificationHandlerFinishesEnqueueAfterCallerCancellation(t *testing.T) {
	t.Parallel()

	pdb, err := testpostgres.New()
	assert.NilError(t, err)
	t.Cleanup(pdb.Cleanup)
	assert.NilError(t, migrations.Up(pdb.ConnStr(), t.Logf))

	db, err := pdb.Open(t.Context())
	assert.NilError(t, err)
	t.Cleanup(db.Close)

	queue := chatqueue.New(db, 1)
	handler := newNotificationHandler(t.Context(), queue)
	message := testWebsocketMessage(time.Now())
	raw, err := json.Marshal(message)
	assert.NilError(t, err)

	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	assert.NilError(t, handler(ctx, raw, message))

	lease, err := queue.Claim(t.Context(), time.Minute)
	assert.NilError(t, err)
	assert.Assert(t, lease != nil)
	assert.Equal(t, lease.ID, "notification")
}

func TestNotificationHandlerRetriesEnqueue(t *testing.T) {
	t.Parallel()

	pdb, err := testpostgres.New()
	assert.NilError(t, err)
	t.Cleanup(pdb.Cleanup)
	assert.NilError(t, migrations.Up(pdb.ConnStr(), t.Logf))

	db, err := pdb.Open(t.Context())
	assert.NilError(t, err)
	t.Cleanup(db.Close)

	queue := chatqueue.New(db, 1)
	handler := newNotificationHandler(t.Context(), queue)
	message := testWebsocketMessage(time.Now())
	raw, err := json.Marshal(message)
	assert.NilError(t, err)

	assert.NilError(t, pdb.Stop())
	done := make(chan error, 1)
	go func() {
		done <- handler(t.Context(), raw, message)
	}()

	select {
	case err := <-done:
		t.Fatalf("enqueue unexpectedly completed while PostgreSQL was stopped: %v", err)
	case <-time.After(250 * time.Millisecond):
	}

	assert.NilError(t, pdb.Start())
	select {
	case err := <-done:
		assert.NilError(t, err)
	case <-time.After(15 * time.Second):
		t.Fatal("enqueue did not recover")
	}

	lease, err := queue.Claim(t.Context(), time.Minute)
	assert.NilError(t, err)
	assert.Assert(t, lease != nil)
	assert.Equal(t, lease.ID, "notification")
}

func testWebsocketMessage(timestamp time.Time) *eventsub.WebsocketMessage {
	return &eventsub.WebsocketMessage{
		Metadata: &eventsub.WebsocketMessageMetadata{
			MessageID:        "notification",
			MessageType:      "notification",
			MessageTimestamp: timestamp,
		},
		Payload: &eventsub.NotificationPayload{
			Subscription: &eventsub.Subscription{
				Type: eventsub.ChatMessageSubscriptionType,
				Condition: &eventsub.ChatMessageSubscriptionCondition{
					UserID: 99,
				},
			},
			Event: &eventsub.ChatMessageEvent{
				MessageID:            "message",
				BroadcasterUserID:    idstr.IDStr(1),
				BroadcasterUserLogin: "channel",
				ChatterUserID:        idstr.IDStr(2),
			},
		},
	}
}
