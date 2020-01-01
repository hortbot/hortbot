package bnsq

import (
	"context"
	atomic "sync/atomic"
	"testing"
	"time"

	"github.com/hortbot/hortbot/internal/pkg/docker/dnsq"
	"github.com/nsqio/go-nsq"
	"gotest.tools/v3/assert"
)

func TestSubscriberBadMessage(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	addr, cleanup, err := dnsq.New()
	assert.NilError(t, err)
	defer cleanup()

	producer, err := nsq.NewProducer(addr, defaultConfig())
	assert.NilError(t, err)
	defer producer.Stop()

	const topic = "topic"

	var count int64
	inc := func(_ *message) error {
		atomic.AddInt64(&count, 1)
		return nil
	}

	subscriber := newSubscriber(addr, topic, "channel")
	go subscriber.run(ctx, inc) //nolint:errcheck

	assert.NilError(t, producer.Publish(topic, []byte("{")))

	time.Sleep(100 * time.Millisecond)

	assert.Equal(t, count, int64(0))
}
