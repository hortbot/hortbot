package bnsq

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hortbot/hortbot/internal/pkg/correlation"
	"github.com/leononame/clock"
	"github.com/nsqio/go-nsq"
	"github.com/zikaeroh/ctxlog"
	"go.uber.org/zap"
)

type publisher struct {
	ready    chan struct{}
	addr     string
	clk      clock.Clock
	config   *nsq.Config
	producer *nsq.Producer
}

func newPublisher(addr string, opts ...PublisherOption) *publisher {
	p := &publisher{
		ready: make(chan struct{}),
		addr:  addr,
	}

	for _, opt := range opts {
		opt.applyToPublisher(p)
	}

	if p.clk == nil {
		p.clk = clock.New()
	}

	if p.config == nil {
		p.config = defaultConfig()
	}

	return p
}

func (p *publisher) run(ctx context.Context) error {
	producer, err := nsq.NewProducer(p.addr, p.config)
	if err != nil {
		ctxlog.Error(ctx, "error creating producer", zap.Error(err))
		return fmt.Errorf("creating producer: %w", err)
	}
	defer producer.Stop()

	producer.SetLogger(nsqLoggerFrom(ctx), nsq.LogLevelInfo)

	p.producer = producer

	if err := producer.Ping(); err != nil {
		ctxlog.Error(ctx, "error pinging server", zap.Error(err))
		return fmt.Errorf("pinging server: %w", err)
	}

	close(p.ready)

	<-ctx.Done()
	return ctx.Err()
}

func (p *publisher) publish(ctx context.Context, topic string, payload any) error {
	select {
	case <-p.ready:
	case <-ctx.Done():
		ctxlog.Error(ctx, "timeout waiting for connection to be ready")
		return ctx.Err()
	}

	pl, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshaling payload: %w", err)
	}

	m := &message{
		Metadata: Metadata{
			Timestamp:   p.clk.Now(),
			Correlation: correlation.FromContext(ctx),
		},
		Payload: pl,
	}

	body, err := json.Marshal(m)
	if err != nil {
		return fmt.Errorf("marshaling message: %w", err)
	}

	doneChan := make(chan *nsq.ProducerTransaction, 1)

	if err := p.producer.PublishAsync(topic, body, doneChan); err != nil {
		return fmt.Errorf("publishing message: %w", err)
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
