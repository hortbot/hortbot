package bnsq_test

import (
	"encoding/json"
	"sync/atomic"
	"testing"
	"time"

	"github.com/hortbot/hortbot/internal/bnsq"
	"github.com/hortbot/hortbot/internal/pkg/docker/dnsq"
	"github.com/hortbot/hortbot/internal/pkg/errgroupx"
	"github.com/nsqio/go-nsq"
	"gotest.tools/v3/assert"
)

func TestEventsubNotify(t *testing.T) {
	t.Parallel()

	addr, cleanup, err := dnsq.New()
	assert.NilError(t, err)
	defer cleanup()

	ctx, cancel := testContext(t)
	defer cancel()

	const channel = "blue"

	received := make(chan *bnsq.EventsubNotify, 10)

	publisher := bnsq.NewEventsubNotifyPublisher(addr)

	subscriber := bnsq.EventsubNotifySubscriber{
		Addr:    addr,
		Channel: channel,
		OnEventsubNotify: func(n *bnsq.EventsubNotify, _ *bnsq.Metadata) error {
			received <- n
			return nil
		},
	}

	g := errgroupx.FromContext(ctx)

	g.Go(publisher.Run)
	g.Go(subscriber.Run)

	assert.NilError(t, publisher.NotifyEventsubUpdates(ctx))
	assert.NilError(t, publisher.NotifyEventsubUpdates(ctx))

	var (
		got1 *bnsq.EventsubNotify
		got2 *bnsq.EventsubNotify
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

	assert.DeepEqual(t, got1, &bnsq.EventsubNotify{})

	assert.DeepEqual(t, got2, &bnsq.EventsubNotify{})

	assert.Equal(t, len(received), 0)
}

func TestEventsubNotifyBadDecode(t *testing.T) {
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
	inc := func(*bnsq.EventsubNotify, *bnsq.Metadata) error {
		count.Add(1)
		return nil
	}

	subscriber := bnsq.EventsubNotifySubscriber{
		Addr:             addr,
		Channel:          channel,
		OnEventsubNotify: inc,
	}

	g := errgroupx.FromContext(ctx)
	g.Go(subscriber.Run)

	b, err := json.Marshal(&bnsq.Message{
		Payload: []byte("true"),
	})
	assert.NilError(t, err)

	assert.NilError(t, producer.Publish(bnsq.EventsubNotifyTopic, b))

	time.Sleep(100 * time.Millisecond)

	g.Stop()
	_ = g.Wait()

	assert.Equal(t, count.Load(), int64(0))
}
