package bnsq_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/hortbot/hortbot/internal/bnsq"
	"github.com/hortbot/hortbot/internal/pkg/docker/dnsq"
	"github.com/hortbot/hortbot/internal/pkg/errgroupx"
	"github.com/leononame/clock"
	"github.com/nsqio/go-nsq"
	"go.uber.org/atomic"
	"gotest.tools/v3/assert"
)

func TestSendMessage(t *testing.T) {
	t.Parallel()

	ctx, cancel := testContext(t)
	defer cancel()

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
	_ = g.Wait()

	assert.DeepEqual(t, got, &bnsq.SendMessage{
		Origin:  origin,
		Target:  "#foobar",
		Message: "this is my message",
	})

	assert.Equal(t, len(received), 0)
}

func TestSendMessageBadAddr(t *testing.T) {
	t.Parallel()

	ctx, cancel := testContext(t)
	defer cancel()

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

	ctx, cancel := testContext(t)
	defer cancel()

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

	ctx, cancel := testContext(t)
	defer cancel()

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
	_ = g.Wait()

	assert.Equal(t, len(received), 0)
}

func TestSendMessageBadDecode(t *testing.T) {
	t.Parallel()

	addr, cleanup, err := dnsq.New()
	assert.NilError(t, err)
	defer cleanup()

	ctx, cancel := testContext(t)
	defer cancel()

	const (
		origin  = "hortbot"
		channel = "blue"
	)

	producer, err := nsq.NewProducer(addr, bnsq.DefaultConfig())
	assert.NilError(t, err)
	producer.SetLogger(bnsq.NsqLoggerFrom(ctx), nsq.LogLevelInfo)
	defer producer.Stop()

	var count atomic.Int64
	inc := func(*bnsq.SendMessage, *bnsq.Metadata) error {
		count.Inc()
		return nil
	}

	subscriber := bnsq.SendMessageSubscriber{
		Addr:          addr,
		Origin:        origin,
		Channel:       channel,
		OnSendMessage: inc,
	}

	g := errgroupx.FromContext(ctx)
	g.Go(subscriber.Run)

	b, err := json.Marshal(&bnsq.Message{
		Payload: []byte("true"),
	})
	assert.NilError(t, err)

	assert.NilError(t, producer.Publish(bnsq.SendMessageTopic+origin, b))

	time.Sleep(100 * time.Millisecond)

	g.Stop()
	_ = g.Wait()

	assert.Equal(t, count.Load(), int64(0))
}
