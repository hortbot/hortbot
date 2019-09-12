package bnsq

import (
	"context"
	"encoding/json"

	"github.com/leononame/clock"
	"github.com/nsqio/go-nsq"
	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
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
		return err
	}
	defer producer.Stop()

	producer.SetLogger(nsqLoggerFrom(ctx), nsq.LogLevelInfo)

	p.producer = producer

	if err := producer.Ping(); err != nil {
		return err
	}

	close(p.ready)

	<-ctx.Done()
	return ctx.Err()
}

func (p *publisher) publish(ctx context.Context, topic string, payload interface{}) error {
	span, ctx := opentracing.StartSpanFromContext(ctx, topic, ext.SpanKindProducer)
	defer span.Finish()

	select {
	case <-p.ready:
	case <-ctx.Done():
		return ctx.Err()
	}

	pl, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	carrier := make(opentracing.TextMapCarrier)
	if err := span.Tracer().Inject(span.Context(), opentracing.TextMap, carrier); err != nil {
		return err
	}

	m := &message{
		Timestamp:    p.clk.Now(),
		TraceCarrier: carrier,
		Payload:      pl,
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
		return pt.Error
	case <-ctx.Done():
		return ctx.Err()
	}
}
