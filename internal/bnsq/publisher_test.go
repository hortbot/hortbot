package bnsq

import (
	"context"
	"testing"
	"time"

	"github.com/hortbot/hortbot/internal/pkg/docker/dnsq"
	"github.com/hortbot/hortbot/internal/pkg/errgroupx"
	"github.com/hortbot/hortbot/internal/pkg/jsonx"
	"github.com/hortbot/hortbot/internal/pkg/testutil"
	"github.com/nsqio/go-nsq"
	"github.com/zikaeroh/ctxlog"
	"gotest.tools/v3/assert"
)

func TestPublishNoRun(t *testing.T) {
	t.Parallel()

	p := newPublisher("localhost:invalid")

	ctx, cancel := testContext(t)
	defer cancel()

	ctx, cancel2 := context.WithTimeout(ctx, 10*time.Millisecond)
	defer cancel2()

	err := p.publish(ctx, "topic", nil)
	assert.Equal(t, err, context.DeadlineExceeded)
}

func TestPublishBadConfig(t *testing.T) {
	t.Parallel()

	badConfig := nsq.NewConfig()
	badConfig.SampleRate = -1
	p := newPublisher("localhost:invalid", WithConfig(badConfig))

	ctx, cancel := testContext(t)
	defer cancel()

	err := p.run(ctx)
	assert.ErrorContains(t, err, "invalid SampleRate")
}

func TestPublishUnmarshalable(t *testing.T) {
	t.Parallel()

	addr, cleanup, err := dnsq.New()
	assert.NilError(t, err)
	defer cleanup()

	ctx, cancel := testContext(t)
	defer cancel()

	publisher := newPublisher(addr)

	g := errgroupx.FromContext(ctx)

	g.Go(publisher.run)

	err = publisher.publish(ctx, "topic", jsonx.Unmarshallable())
	assert.ErrorContains(t, err, jsonx.ErrUnmarshallable.Error())

	g.Stop()
	_ = g.Wait()
}

func TestPublishNotConnected(t *testing.T) {
	t.Parallel()

	addr, cleanup, err := dnsq.New()
	assert.NilError(t, err)

	ctx, cancel := testContext(t)
	defer cancel()

	publisher := newPublisher(addr)

	g := errgroupx.FromContext(ctx)

	g.Go(publisher.run)

	err = publisher.publish(ctx, "topic", "some value")
	assert.NilError(t, err)

	time.Sleep(100 * time.Millisecond)
	cleanup()

	err = publisher.publish(ctx, "topic", "some value")
	assert.Assert(t, err != nil)

	g.Stop()
	_ = g.Wait()
}

func testContext(t testing.TB) (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)

	logger := testutil.Logger(t)
	ctx = ctxlog.WithLogger(ctx, logger)

	return ctx, cancel
}
