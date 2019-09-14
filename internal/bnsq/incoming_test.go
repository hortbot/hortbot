package bnsq_test

import (
	"context"
	"testing"
	"time"

	"github.com/fortytw2/leaktest"
	"github.com/hortbot/hortbot/internal/bnsq"
	"github.com/hortbot/hortbot/internal/pkg/ctxlog"
	"github.com/hortbot/hortbot/internal/pkg/errgroupx"
	"github.com/hortbot/hortbot/internal/pkg/ircx"
	"github.com/hortbot/hortbot/internal/pkg/testutil"
	"github.com/hortbot/hortbot/internal/pkg/testutil/nsqtest"
	"go.opencensus.io/trace"
	"gotest.tools/v3/assert"
)

func TestIncoming(t *testing.T) {
	defer leaktest.Check(t)()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	logger := testutil.Logger(t)
	ctx = ctxlog.WithLogger(ctx, logger)

	addr, cleanup, err := nsqtest.New()
	assert.NilError(t, err)
	defer cleanup()

	const channel = "blue"

	received := make(chan *bnsq.Incoming, 10)

	publisher := bnsq.NewIncomingPublisher(addr)

	subscriber := bnsq.IncomingSubscriber{
		Addr:    addr,
		Channel: channel,
		OnIncoming: func(n *bnsq.Incoming, _ trace.SpanContext) error {
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

	got1 := <-received
	got2 := <-received

	got1.Message.Raw = ""
	got2.Message.Raw = ""

	g.Stop()

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
