package bnsq_test

import (
	"context"
	"testing"
	"time"

	"github.com/fortytw2/leaktest"
	"github.com/hortbot/hortbot/internal/bnsq"
	"github.com/hortbot/hortbot/internal/pkg/ctxlog"
	"github.com/hortbot/hortbot/internal/pkg/errgroupx"
	"github.com/hortbot/hortbot/internal/pkg/testutil"
	"github.com/hortbot/hortbot/internal/pkg/testutil/nsqtest"
	"github.com/leononame/clock"
	"gotest.tools/v3/assert"
)

func TestSendMessage(t *testing.T) {
	defer leaktest.Check(t)()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	logger := testutil.Logger(t)
	ctx = ctxlog.WithLogger(ctx, logger)

	addr, cleanup, err := nsqtest.New()
	assert.NilError(t, err)
	defer cleanup()

	clk := clock.NewMock()

	const (
		origin  = "hortbot"
		channel = "blue"
	)

	received := make(chan *bnsq.SendMessage, 10)

	producer := bnsq.NewSendMessageProducer(addr, clk)

	consumer := bnsq.SendMessageConsumer{
		Addr:    addr,
		Origin:  origin,
		Channel: channel,
		OnSendMessage: func(m *bnsq.SendMessage) {
			received <- m
		},
	}

	g := errgroupx.FromContext(ctx)

	g.Go(producer.Run)
	g.Go(consumer.Run)

	assert.NilError(t, producer.SendMessage(origin, "#foobar", "this is my message"))
	assert.NilError(t, producer.SendMessage("wrong", "#other", "nobody can read this"))

	got := <-received

	g.Stop()

	assert.DeepEqual(t, got, &bnsq.SendMessage{
		Timestamp: clk.Now(),
		Origin:    origin,
		Target:    "#foobar",
		Message:   "this is my message",
	})

	assert.Equal(t, len(received), 0)
}

func TestSendMessageBadAddr(t *testing.T) {
	defer leaktest.Check(t)()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	logger := testutil.Logger(t)
	ctx = ctxlog.WithLogger(ctx, logger)

	addr := "localhost:9999"

	clk := clock.NewMock()

	const (
		origin  = "hortbot"
		channel = "blue"
	)

	received := make(chan *bnsq.SendMessage, 10)

	producer := bnsq.NewSendMessageProducer(addr, clk)

	consumer := bnsq.SendMessageConsumer{
		Addr:    addr,
		Origin:  origin,
		Channel: channel,
		OnSendMessage: func(m *bnsq.SendMessage) {
			received <- m
		},
	}

	assert.ErrorContains(t, producer.Run(ctx), "connection refused")
	assert.ErrorContains(t, consumer.Run(ctx), "connection refused")

	assert.Equal(t, len(received), 0)
}

func TestSendMessageConsumerBadChannel(t *testing.T) {
	defer leaktest.Check(t)()

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

	consumer := bnsq.SendMessageConsumer{
		Addr:    addr,
		Origin:  origin,
		Channel: channel,
		OnSendMessage: func(m *bnsq.SendMessage) {
			received <- m
		},
	}

	assert.ErrorContains(t, consumer.Run(ctx), "invalid channel name")

	assert.Equal(t, len(received), 0)
}
