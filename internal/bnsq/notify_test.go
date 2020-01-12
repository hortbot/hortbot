package bnsq_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/hortbot/hortbot/internal/bnsq"
	"github.com/hortbot/hortbot/internal/pkg/correlation"
	"github.com/hortbot/hortbot/internal/pkg/docker/dnsq"
	"github.com/hortbot/hortbot/internal/pkg/errgroupx"
	"github.com/nsqio/go-nsq"
	"github.com/rs/xid"
	"go.opencensus.io/trace"
	"go.uber.org/atomic"
	"gotest.tools/v3/assert"
)

func TestNotify(t *testing.T) {
	t.Parallel()

	addr, cleanup, err := dnsq.New()
	assert.NilError(t, err)
	defer cleanup()

	ctx, cancel := testContext(t)
	defer cancel()

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
	_ = g.Wait()

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

	ctx, cancel := testContext(t)
	defer cancel()

	const (
		botName = "hortbot"
		channel = "blue"
	)

	producer, err := nsq.NewProducer(addr, bnsq.DefaultConfig())
	assert.NilError(t, err)
	producer.SetLogger(bnsq.NsqLoggerFrom(ctx), nsq.LogLevelInfo)
	defer producer.Stop()

	var count atomic.Int64
	inc := func(*bnsq.ChannelUpdatesNotification, *bnsq.Metadata) error {
		count.Inc()
		return nil
	}

	subscriber := bnsq.NotifySubscriber{
		Addr:                   addr,
		BotName:                botName,
		Channel:                channel,
		OnNotifyChannelUpdates: inc,
	}

	g := errgroupx.FromContext(ctx)
	g.Go(subscriber.Run)

	b, err := json.Marshal(&bnsq.Message{
		Payload: []byte("true"),
	})
	assert.NilError(t, err)

	assert.NilError(t, producer.Publish(bnsq.NotifyChannelUpdatesTopic+botName, b))

	time.Sleep(100 * time.Millisecond)

	g.Stop()
	_ = g.Wait()

	assert.Equal(t, count.Load(), int64(0))
}
