package bnsq_test

import (
	"context"
	json "encoding/json"
	atomic "sync/atomic"
	"testing"
	"time"

	"github.com/hortbot/hortbot/internal/bnsq"
	"github.com/hortbot/hortbot/internal/pkg/correlation"
	"github.com/hortbot/hortbot/internal/pkg/ctxlog"
	"github.com/hortbot/hortbot/internal/pkg/docker/dnsq"
	"github.com/hortbot/hortbot/internal/pkg/errgroupx"
	"github.com/hortbot/hortbot/internal/pkg/testutil"
	"github.com/nsqio/go-nsq"
	"github.com/rs/xid"
	"go.opencensus.io/trace"
	"gotest.tools/v3/assert"
)

func TestNotify(t *testing.T) {
	t.Parallel()

	addr, cleanup, err := dnsq.New()
	assert.NilError(t, err)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	logger := testutil.Logger(t)
	ctx = ctxlog.WithLogger(ctx, logger)

	const (
		botName = "hortbot"
		channel = "blue"
	)

	received := make(chan *bnsq.ChannelUpdatesNotification, 10)
	cids := make(chan xid.ID, 10)
	spans := make(chan trace.SpanContext)

	publisher := bnsq.NewNotifyPublisher(addr)

	subscriber := bnsq.NotifySubscriber{
		Addr:    addr,
		BotName: botName,
		Channel: channel,
		OnNotifyChannelUpdates: func(n *bnsq.ChannelUpdatesNotification, m *bnsq.Metadata) error {
			received <- n
			cids <- correlation.FromContext(m.Correlate(context.Background()))
			spans <- m.ParentSpan()
			return nil
		},
	}

	g := errgroupx.FromContext(ctx)

	g.Go(publisher.Run)
	g.Go(subscriber.Run)

	id := xid.New()
	ctx, span := trace.StartSpan(ctx, "TwitchToken")

	assert.NilError(t, publisher.NotifyChannelUpdates(correlation.WithID(ctx, id), botName))
	assert.NilError(t, publisher.NotifyChannelUpdates(ctx, "wrong"))

	got := <-received
	gotID := <-cids
	gotSpan := <-spans

	g.Stop()

	assert.Equal(t, len(received), 0)

	assert.DeepEqual(t, got, &bnsq.ChannelUpdatesNotification{
		BotName: botName,
	})

	assert.Equal(t, gotID, id)
	assert.Equal(t, gotSpan.TraceID, span.SpanContext().TraceID)
}

func TestNotifyBadDecode(t *testing.T) {
	t.Parallel()

	addr, cleanup, err := dnsq.New()
	assert.NilError(t, err)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	const (
		botName = "hortbot"
		channel = "blue"
	)

	producer, err := nsq.NewProducer(addr, bnsq.DefaultConfig())
	assert.NilError(t, err)
	defer producer.Stop()

	var count int64
	inc := func(*bnsq.ChannelUpdatesNotification, *bnsq.Metadata) error {
		atomic.AddInt64(&count, 1)
		return nil
	}

	subscriber := bnsq.NotifySubscriber{
		Addr:                   addr,
		BotName:                botName,
		Channel:                channel,
		OnNotifyChannelUpdates: inc,
	}

	go subscriber.Run(ctx) //nolint:errcheck

	b, err := json.Marshal(&bnsq.Message{
		Payload: []byte("true"),
	})
	assert.NilError(t, err)

	assert.NilError(t, producer.Publish(bnsq.NotifyChannelUpdatesTopic+botName, b))

	time.Sleep(100 * time.Millisecond)

	assert.Equal(t, count, int64(0))
}
