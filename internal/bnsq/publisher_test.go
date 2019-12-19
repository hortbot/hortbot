package bnsq

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/hortbot/hortbot/internal/pkg/ctxlog"
	"github.com/hortbot/hortbot/internal/pkg/docker/dnsq"
	"github.com/hortbot/hortbot/internal/pkg/errgroupx"
	"github.com/hortbot/hortbot/internal/pkg/testutil"
	"github.com/nsqio/go-nsq"
	"gotest.tools/v3/assert"
)

func TestPublishNoRun(t *testing.T) {
	p := newPublisher("localhost:invalid")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	err := p.publish(ctx, "topic", nil)
	assert.Equal(t, err, context.DeadlineExceeded)
}

func TestPublishBadConfig(t *testing.T) {
	badConfig := nsq.NewConfig()
	badConfig.SampleRate = -1
	p := newPublisher("localhost:invalid", PublisherConfig(badConfig))

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	err := p.run(ctx)
	assert.ErrorContains(t, err, "invalid SampleRate")
}

func TestPublishUnmarshallable(t *testing.T) {
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

	publisher := newPublisher(addr)

	g := errgroupx.FromContext(ctx)

	g.Go(publisher.run)

	err = publisher.publish(ctx, "topic", unmarshallable{})
	assert.ErrorContains(t, err, "unmarshallable")

	g.Stop()
}

type unmarshallable struct{}

func (unmarshallable) MarshalJSON() ([]byte, error) {
	return nil, errors.New("unmarshallable")
}
