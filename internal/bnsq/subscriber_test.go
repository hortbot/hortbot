package bnsq

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/hortbot/hortbot/internal/pkg/docker/dnsq"
	"github.com/hortbot/hortbot/internal/pkg/errgroupx"
	"github.com/nsqio/go-nsq"
	"gotest.tools/v3/assert"
)

func TestSubscriberBadMessage(t *testing.T) {
	t.Parallel()

	ctx, cancel := testContext(t)
	defer cancel()

	addr, cleanup, err := dnsq.New()
	assert.NilError(t, err)
	defer cleanup()

	producer, err := nsq.NewProducer(addr, defaultConfig())
	assert.NilError(t, err)
	producer.SetLogger(NsqLoggerFrom(ctx), nsq.LogLevelInfo)
	defer producer.Stop()

	const topic = "topic"

	var count atomic.Int64
	inc := func(_ *message) error {
		count.Add(1)
		return nil
	}

	subscriber := newSubscriber(addr, topic, "channel")

	g := errgroupx.FromContext(ctx)
	g.Go(func(ctx context.Context) error {
		return subscriber.run(ctx, inc)
	})

	assert.NilError(t, producer.Publish(topic, []byte("{")))

	time.Sleep(100 * time.Millisecond)

	assert.Equal(t, count.Load(), int64(0))

	g.Stop()
	_ = g.Wait()
}
