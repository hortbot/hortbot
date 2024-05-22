package bnsq_test

import (
	"encoding/json"
	"sync/atomic"
	"testing"
	"time"

	"github.com/hortbot/hortbot/internal/bnsq"
	"github.com/hortbot/hortbot/internal/pkg/apiclient/twitch/eventsub"
	"github.com/hortbot/hortbot/internal/pkg/docker/dnsq"
	"github.com/hortbot/hortbot/internal/pkg/errgroupx"
	"github.com/nsqio/go-nsq"
	"gotest.tools/v3/assert"
)

func TestIncomingWebsocketMessage(t *testing.T) {
	t.Parallel()

	addr, cleanup, err := dnsq.New()
	assert.NilError(t, err)
	defer cleanup()

	ctx, cancel := testContext(t)
	defer cancel()

	const channel = "blue"

	received := make(chan *bnsq.IncomingWebsocketMessage, 10)

	publisher := bnsq.NewIncomingWebsocketMessagePublisher(addr)

	subscriber := bnsq.IncomingWebsocketMessageSubscriber{
		Addr:    addr,
		Channel: channel,
		OnIncomingWebsocketMessage: func(n *bnsq.IncomingWebsocketMessage, _ *bnsq.Metadata) error {
			received <- n
			return nil
		},
	}

	g := errgroupx.FromContext(ctx)

	g.Go(publisher.Run)
	g.Go(subscriber.Run)

	m1 := &eventsub.WebsocketMessage{
		Metadata: &eventsub.WebsocketMessageMetadata{
			MessageType: "session_welcome",
		},
		Payload: &eventsub.SessionWelcomePayload{
			Session: eventsub.Session{
				ID: "session-id",
			},
		},
	}
	m2 := &eventsub.WebsocketMessage{
		Metadata: &eventsub.WebsocketMessageMetadata{
			MessageType: "session_keepalive",
		},
		Payload: &eventsub.SessionKeepalivePayload{},
	}

	assert.NilError(t, publisher.Publish(ctx, m1))
	assert.NilError(t, publisher.Publish(ctx, m2))

	var (
		got1 *bnsq.IncomingWebsocketMessage
		got2 *bnsq.IncomingWebsocketMessage
	)

	select {
	case got1 = <-received:
	case <-ctx.Done():
		assert.NilError(t, ctx.Err())
	}

	select {
	case got2 = <-received:
	case <-ctx.Done():
		assert.NilError(t, ctx.Err())
	}

	g.Stop()
	_ = g.Wait()

	assert.DeepEqual(t, got1, &bnsq.IncomingWebsocketMessage{
		Message: m1,
	})

	assert.DeepEqual(t, got2, &bnsq.IncomingWebsocketMessage{
		Message: m2,
	})

	assert.Equal(t, len(received), 0)
}

func TestIncomingWebsocketMessageBadDecode(t *testing.T) {
	t.Parallel()

	addr, cleanup, err := dnsq.New()
	assert.NilError(t, err)
	defer cleanup()

	ctx, cancel := testContext(t)
	defer cancel()

	const channel = "blue"

	producer, err := nsq.NewProducer(addr, bnsq.DefaultConfig())
	assert.NilError(t, err)
	producer.SetLogger(bnsq.NsqLoggerFrom(ctx), nsq.LogLevelInfo)
	defer producer.Stop()

	var count atomic.Int64
	inc := func(*bnsq.IncomingWebsocketMessage, *bnsq.Metadata) error {
		count.Add(1)
		return nil
	}

	subscriber := bnsq.IncomingWebsocketMessageSubscriber{
		Addr:                       addr,
		Channel:                    channel,
		OnIncomingWebsocketMessage: inc,
	}

	g := errgroupx.FromContext(ctx)
	g.Go(subscriber.Run)

	b, err := json.Marshal(&bnsq.Message{
		Payload: []byte("true"),
	})
	assert.NilError(t, err)

	assert.NilError(t, producer.Publish(bnsq.IncomingWebsocketMessageTopic, b))

	time.Sleep(100 * time.Millisecond)

	g.Stop()
	_ = g.Wait()

	assert.Equal(t, count.Load(), int64(0))
}
