package bnsq

import (
	"context"
	"encoding/json"

	"github.com/leononame/clock"
	"github.com/nsqio/go-nsq"
)

type publisher struct {
	addr     string
	clk      clock.Clock
	config   *nsq.Config
	producer *nsq.Producer
	ready    chan struct{}
}

type PublisherOption func(*publisher)

func newPublisher(addr string, opts ...PublisherOption) *publisher {
	p := &publisher{
		addr:  addr,
		ready: make(chan struct{}),
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
	close(p.ready)

	if err := producer.Ping(); err != nil {
		return err
	}

	<-ctx.Done()
	return ctx.Err()
}

func (p *publisher) publish(topic string, payload interface{}) error {
	<-p.ready

	m, err := newMessage(payload, p.clk)
	if err != nil {
		return err
	}

	body, err := json.Marshal(m)
	if err != nil {
		return err
	}

	return p.producer.Publish(topic, body)
}
