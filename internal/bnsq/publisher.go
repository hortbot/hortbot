package bnsq

import (
	"context"
	"encoding/json"
	"os"
	"runtime"

	"github.com/hortbot/hortbot/internal/pkg/correlation"
	"github.com/hortbot/hortbot/internal/pkg/ctxlog"
	"github.com/leononame/clock"
	"github.com/nsqio/go-nsq"
	"go.opencensus.io/trace"
	"go.opencensus.io/trace/propagation"
	"go.uber.org/zap"
)

type publisher struct {
	ready    chan struct{}
	addr     string
	clk      clock.Clock
	config   *nsq.Config
	producer *nsq.Producer
}

type PublisherOption func(*publisher)

func newPublisher(addr string, opts ...PublisherOption) *publisher {
	p := &publisher{
		ready: make(chan struct{}),
		addr:  addr,
	}

	for _, opt := range opts {
		opt(p)
	}

	if p.clk == nil {
		p.clk = clock.New()
	}

	if p.config == nil {
		p.config = defaultConfig()
	}

	return p
}

func PublisherClock(clk clock.Clock) PublisherOption {
	return func(p *publisher) {
		p.clk = clk
	}
}

func PublisherConfig(config *nsq.Config) PublisherOption {
	return func(p *publisher) {
		p.config = config
	}
}

func (p *publisher) run(ctx context.Context) error {
	producer, err := nsq.NewProducer(p.addr, p.config)
	if err != nil {
		ctxlog.Error(ctx, "error creating producer", zap.Error(err))
		return err
	}
	defer producer.Stop()

	producer.SetLogger(nsqLoggerFrom(ctx), nsq.LogLevelDebug) // TODO: Revert level change after Github Actions deadlock is found.

	p.producer = producer

	ctxlog.Info(ctx, "pinging producer") // TODO: Remove me after Github Actions deadlock is found.
	if err := producer.Ping(); err != nil {
		ctxlog.Error(ctx, "error pinging server", zap.Error(err))
		return err
	}

	ctxlog.Info(ctx, "producer ready") // TODO: Remove me after Github Actions deadlock is found.
	close(p.ready)

	<-ctx.Done()
	err = ctx.Err()
	ctxlog.Info(ctx, "producer exiting", zap.Error(err)) // TODO: Remove me after Github Actions deadlock is found.
	return err
}

func (p *publisher) publish(ctx context.Context, topic string, payload interface{}) error {
	ctx, span := trace.StartSpan(ctx, topic)
	defer span.End()

	select {
	case <-p.ready:
	case <-ctx.Done():
		ctxlog.Error(ctx, "timeout waiting for connection to be ready")

		// TODO: Remove me after Github Actions deadlock is found.
		stack := func() []byte {
			buf := make([]byte, 1024)
			for {
				n := runtime.Stack(buf, true)
				if n < len(buf) {
					return buf[:n]
				}
				buf = make([]byte, 2*len(buf))
			}
		}

		os.Stderr.Write(stack())

		return ctx.Err()
	}

	pl, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	m := &message{
		Metadata: Metadata{
			Timestamp:   p.clk.Now(),
			TraceSpan:   propagation.Binary(span.SpanContext()),
			Correlation: correlation.FromContext(ctx),
		},
		Payload: pl,
	}

	body, err := json.Marshal(m)
	if err != nil {
		return err
	}

	doneChan := make(chan *nsq.ProducerTransaction, 1)

	if err := p.producer.PublishAsync(topic, body, doneChan); err != nil {
		return err
	}

	select {
	case pt := <-doneChan:
		if err := pt.Error; err != nil {
			ctxlog.Error(ctx, "producer transaction error", zap.Error(err))
			return err
		}
		metricPublished.WithLabelValues(topic).Inc()
		return nil
	case <-ctx.Done():
		ctxlog.Error(ctx, "timeout waiting for async publish completion")
		return ctx.Err()
	}
}
