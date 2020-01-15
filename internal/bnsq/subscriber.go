package bnsq

import (
	"context"
	"encoding/json"
	"time"

	"github.com/hortbot/hortbot/internal/pkg/ctxlog"
	"github.com/leononame/clock"
	"github.com/nsqio/go-nsq"
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

func (s *subscriber) run(ctx context.Context, fn func(m *message) error) error {
	consumer, err := nsq.NewConsumer(s.topic, s.channel, s.config)
	if err != nil {
		ctxlog.Error(ctx, "error creating consumer", zap.Error(err))
		return err
	}
	defer func() {
		consumer.Stop()
		<-consumer.StopChan
	}()

	consumer.SetLogger(nsqLoggerFrom(ctx), nsq.LogLevelInfo)

	consumer.AddHandler(nsq.HandlerFunc(func(msg *nsq.Message) error {
		if dur := testingSleep.Load(); dur != 0 {
			s.clk.Sleep(dur)
		}

		m := &message{}

		if err := json.Unmarshal(msg.Body, m); err != nil {
			ctxlog.Warn(ctx, "error decoding message", zap.Error(err))
			return nil
		}

		if s.maxAge > 0 {
			since := s.clk.Since(m.Metadata.Timestamp)
			if since > s.maxAge {
				ctxlog.Warn(ctx, "message too old, dropping", zap.Duration("since", since))
				return nil
			}
		}

		defer metricHandled.WithLabelValues(s.topic).Inc()

		return fn(m)
	}))

	if err := consumer.ConnectToNSQD(s.addr); err != nil {
		ctxlog.Error(ctx, "error connecting to server", zap.Error(err))
		return err
	}

	<-ctx.Done()
	return ctx.Err()
}
