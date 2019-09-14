package bnsq

import (
	"context"
	"encoding/json"
	"time"

	"github.com/hortbot/hortbot/internal/pkg/ctxlog"
	"github.com/leononame/clock"
	"github.com/nsqio/go-nsq"
	"go.opencensus.io/trace"
	"go.opencensus.io/trace/propagation"
	"go.uber.org/zap"
)

type subscriber struct {
	addr    string
	topic   string
	channel string
	clk     clock.Clock
	config  *nsq.Config
	maxAge  time.Duration
}

type SubscriberOption func(*subscriber)

func newSubscriber(addr string, topic string, channel string, opts ...SubscriberOption) *subscriber {
	s := &subscriber{
		addr:    addr,
		topic:   topic,
		channel: channel,
	}

	for _, opt := range opts {
		opt(s)
	}

	if s.clk == nil {
		s.clk = clock.New()
	}

	if s.config == nil {
		s.config = defaultConfig()
	}

	return s
}

func SubscriberClock(clk clock.Clock) SubscriberOption {
	return func(s *subscriber) {
		s.clk = clk
	}
}

func SubscriberConfig(config *nsq.Config) SubscriberOption {
	return func(s *subscriber) {
		s.config = config
	}
}

func SubscriberMaxAge(d time.Duration) SubscriberOption {
	return func(s *subscriber) {
		s.maxAge = d
	}
}

func (s *subscriber) run(ctx context.Context, fn func(m *message, parent trace.SpanContext) error) error {
	logger := ctxlog.FromContext(ctx)

	consumer, err := nsq.NewConsumer(s.topic, s.channel, s.config)
	if err != nil {
		return err
	}
	defer consumer.Stop()

	consumer.SetLogger(nsqLoggerFrom(ctx), nsq.LogLevelInfo)

	consumer.AddHandler(nsq.HandlerFunc(func(msg *nsq.Message) error {
		if testingSleep != 0 {
			s.clk.Sleep(testingSleep)
		}

		m := &message{}

		if err := json.Unmarshal(msg.Body, m); err != nil {
			return nil
		}

		parent, _ := propagation.FromBinary(m.TraceSpan)

		if s.maxAge > 0 {
			since := s.clk.Since(m.Timestamp)
			if since > s.maxAge {
				logger.Warn("message too old, dropping", zap.Duration("since", since))
				return nil
			}
		}

		return fn(m, parent)
	}))

	if err := consumer.ConnectToNSQD(s.addr); err != nil {
		return err
	}

	<-ctx.Done()
	return ctx.Err()
}
