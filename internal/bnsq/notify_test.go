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
	"github.com/opentracing/opentracing-go"
	"gotest.tools/v3/assert"
)

func TestNotify(t *testing.T) {
	defer leaktest.Check(t)()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	logger := testutil.Logger(t)
	ctx = ctxlog.WithLogger(ctx, logger)

	addr, cleanup, err := nsqtest.New()
	assert.NilError(t, err)
	defer cleanup()

	const (
		botName = "hortbot"
		channel = "blue"
	)

	received := make(chan *bnsq.ChannelUpdatesNotification, 10)

	publisher := bnsq.NewNotifyPublisher(addr)

	subscriber := bnsq.NotifySubscriber{
		Addr:    addr,
		BotName: botName,
		Channel: channel,
		OnNotifyChannelUpdates: func(n *bnsq.ChannelUpdatesNotification, _ opentracing.SpanReference) {
			received <- n
		},
	}

	g := errgroupx.FromContext(ctx)

	g.Go(publisher.Run)
	g.Go(subscriber.Run)

	assert.NilError(t, publisher.NotifyChannelUpdates(ctx, botName))
	assert.NilError(t, publisher.NotifyChannelUpdates(ctx, "wrong"))

	got := <-received

	g.Stop()

	assert.DeepEqual(t, got, &bnsq.ChannelUpdatesNotification{
		BotName: botName,
	})

	assert.Equal(t, len(received), 0)
}
