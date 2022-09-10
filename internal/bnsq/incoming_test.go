package bnsq_test

import (
	"context"
	"encoding/json"
	"sync/atomic"
	"testing"
	"time"

	"github.com/hortbot/hortbot/internal/bnsq"
	"github.com/hortbot/hortbot/internal/pkg/docker/dnsq"
	"github.com/hortbot/hortbot/internal/pkg/errgroupx"
	"github.com/hortbot/hortbot/internal/pkg/ircx"
	"github.com/hortbot/hortbot/internal/pkg/testutil"
	"github.com/nsqio/go-nsq"
	"github.com/zikaeroh/ctxlog"
	"gotest.tools/v3/assert"
)

func TestIncoming(t *testing.T) {
	t.Parallel()

	addr, cleanup, err := dnsq.New()
	assert.NilError(t, err)
	defer cleanup()

	ctx, cancel := testContext(t)
	defer cancel()

	const channel = "blue"

	received := make(chan *bnsq.Incoming, 10)

	publisher := bnsq.NewIncomingPublisher(addr)

	subscriber := bnsq.IncomingSubscriber{
		Addr:    addr,
		Channel: channel,
		OnIncoming: func(n *bnsq.Incoming, _ *bnsq.Metadata) error {
			received <- n
			return nil
		},
	}

	g := errgroupx.FromContext(ctx)

	g.Go(publisher.Run)
	g.Go(subscriber.Run)

	m1 := ircx.PrivMsg("#foobar", "test")
	m2 := ircx.PrivMsg("#someone", "other test")

	assert.NilError(t, publisher.Publish(ctx, "hortbot", m1))
	assert.NilError(t, publisher.Publish(ctx, "otherbot", m2))

	var (
		got1 *bnsq.Incoming
		got2 *bnsq.Incoming
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

	got1.Message.Raw = ""
	got2.Message.Raw = ""

	g.Stop()
	_ = g.Wait()

	assert.DeepEqual(t, got1, &bnsq.Incoming{
		Origin:  "hortbot",
		Message: m1,
	})

	assert.DeepEqual(t, got2, &bnsq.Incoming{
		Origin:  "otherbot",
		Message: m2,
	})

	assert.Equal(t, len(received), 0)
}

func TestIncomingBadDecode(t *testing.T) {
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
	inc := func(*bnsq.Incoming, *bnsq.Metadata) error {
		count.Add(1)
		return nil
	}

	subscriber := bnsq.IncomingSubscriber{
		Addr:       addr,
		Channel:    channel,
		OnIncoming: inc,
	}

	g := errgroupx.FromContext(ctx)
	g.Go(subscriber.Run)

	b, err := json.Marshal(&bnsq.Message{
		Payload: []byte("true"),
	})
	assert.NilError(t, err)

	assert.NilError(t, producer.Publish(bnsq.IncomingTopic, b))

	time.Sleep(100 * time.Millisecond)

	g.Stop()
	_ = g.Wait()

	assert.Equal(t, count.Load(), int64(0))
}

func testContext(t testing.TB) (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)

	logger := testutil.Logger(t)
	ctx = ctxlog.WithLogger(ctx, logger)

	return ctx, cancel
}
