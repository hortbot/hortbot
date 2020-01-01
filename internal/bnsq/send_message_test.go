package bnsq_test

import (
	"context"
	json "encoding/json"
	atomic "sync/atomic"
	"testing"
	"time"

	"github.com/hortbot/hortbot/internal/bnsq"
	"github.com/hortbot/hortbot/internal/pkg/ctxlog"
	"github.com/hortbot/hortbot/internal/pkg/docker/dnsq"
	"github.com/hortbot/hortbot/internal/pkg/errgroupx"
	"github.com/hortbot/hortbot/internal/pkg/testutil"
	"github.com/leononame/clock"
	"github.com/nsqio/go-nsq"
	"gotest.tools/v3/assert"
)

func TestSendMessage(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	logger := testutil.Logger(t)
	ctx = ctxlog.WithLogger(ctx, logger)

	addr, cleanup, err := dnsq.New()
	assert.NilError(t, err)
	defer cleanup()

	clk := clock.NewMock()

	const (
		origin  = "hortbot"
		channel = "blue"
	)

	received := make(chan *bnsq.SendMessage, 10)

	publisher := bnsq.NewSendMessagePublisher(addr, bnsq.PublisherClock(clk))

	subscriber := bnsq.SendMessageSubscriber{
		Addr:    addr,
		Origin:  origin,
		Channel: channel,
		Opts: []bnsq.SubscriberOption{
			bnsq.SubscriberClock(clk),
		},
		OnSendMessage: func(m *bnsq.SendMessage, _ *bnsq.Metadata) error {
			received <- m
			return nil
		},
	}

	g := errgroupx.FromContext(ctx)

	g.Go(publisher.Run)
	g.Go(subscriber.Run)

	assert.NilError(t, publisher.SendMessage(ctx, origin, "#foobar", "this is my message"))
	assert.NilError(t, publisher.SendMessage(ctx, "wrong", "#other", "nobody can read this"))

	got := <-received

	g.Stop()

	assert.DeepEqual(t, got, &bnsq.SendMessage{
		Origin:  origin,
		Target:  "#foobar",
		Message: "this is my message",
	})

	assert.Equal(t, len(received), 0)
}

func TestSendMessageBadAddr(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	logger := testutil.Logger(t)
	ctx = ctxlog.WithLogger(ctx, logger)

	addr := "localhost:9999"
	config := nsq.NewConfig()

	const (
		origin  = "hortbot"
		channel = "blue"
	)

	received := make(chan *bnsq.SendMessage, 10)

	publisher := bnsq.NewSendMessagePublisher(addr, bnsq.PublisherConfig(config))

	subscriber := bnsq.SendMessageSubscriber{
		Addr:    addr,
		Origin:  origin,
		Channel: channel,
		Opts: []bnsq.SubscriberOption{
			bnsq.SubscriberConfig(config),
		},
		OnSendMessage: func(m *bnsq.SendMessage, _ *bnsq.Metadata) error {
			received <- m
			return nil
		},
	}

	assert.ErrorContains(t, publisher.Run(ctx), "connection refused")
	assert.ErrorContains(t, subscriber.Run(ctx), "connection refused")

	assert.Equal(t, len(received), 0)
}

func TestSendMessageSubscriberBadChannel(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	logger := testutil.Logger(t)
	ctx = ctxlog.WithLogger(ctx, logger)

	addr := "localhost:9999"

	const (
		origin  = "hortbot"
		channel = "$asdasdno/.asda()^&%"
	)

	received := make(chan *bnsq.SendMessage, 10)

	subscriber := bnsq.SendMessageSubscriber{
		Addr:    addr,
		Origin:  origin,
		Channel: channel,
		OnSendMessage: func(m *bnsq.SendMessage, _ *bnsq.Metadata) error {
			received <- m
			return nil
		},
	}

	assert.ErrorContains(t, subscriber.Run(ctx), "invalid channel name")

	assert.Equal(t, len(received), 0)
}

func TestMaxAge(t *testing.T) {
	// Must not be parallel due to the global variable modification below.

	addr, cleanup, err := dnsq.New()
	assert.NilError(t, err)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	logger := testutil.Logger(t)
	ctx = ctxlog.WithLogger(ctx, logger)

	clk := clock.NewMock()

	const (
		origin  = "hortbot"
		channel = "blue"
	)

	received := make(chan *bnsq.SendMessage, 10)

	publisher := bnsq.NewSendMessagePublisher(addr, bnsq.PublisherClock(clk))

	subscriber := bnsq.SendMessageSubscriber{
		Addr:    addr,
		Origin:  origin,
		Channel: channel,
		Opts: []bnsq.SubscriberOption{
			bnsq.SubscriberClock(clk),
			bnsq.SubscriberMaxAge(30 * time.Second),
		},
		OnSendMessage: func(m *bnsq.SendMessage, _ *bnsq.Metadata) error {
			received <- m
			return nil
		},
	}

	bnsq.TestingSleep(time.Minute)
	defer bnsq.TestingSleep(0)

	g := errgroupx.FromContext(ctx)

	g.Go(publisher.Run)
	g.Go(subscriber.Run)

	time.Sleep(10 * time.Millisecond)
	assert.NilError(t, publisher.SendMessage(ctx, origin, "#foobar", "this is my message"))

	time.Sleep(10 * time.Millisecond)
	clk.Forward(2 * time.Minute)
	time.Sleep(10 * time.Millisecond)

	g.Stop()

	assert.Equal(t, len(received), 0)
}

func TestSendMessageBadDecode(t *testing.T) {
	t.Parallel()

	addr, cleanup, err := dnsq.New()
	assert.NilError(t, err)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	const (
		origin  = "hortbot"
		channel = "blue"
	)

	producer, err := nsq.NewProducer(addr, bnsq.DefaultConfig())
	assert.NilError(t, err)
	defer producer.Stop()

	var count int64
	inc := func(*bnsq.SendMessage, *bnsq.Metadata) error {
		atomic.AddInt64(&count, 1)
		return nil
	}

	subscriber := bnsq.SendMessageSubscriber{
		Addr:          addr,
		Origin:        origin,
		Channel:       channel,
		OnSendMessage: inc,
	}

	go subscriber.Run(ctx) //nolint:errcheck

	b, err := json.Marshal(&bnsq.Message{
		Payload: []byte("true"),
	})
	assert.NilError(t, err)

	assert.NilError(t, producer.Publish(bnsq.SendMessageTopic+origin, b))

	time.Sleep(100 * time.Millisecond)

	assert.Equal(t, count, int64(0))
}
