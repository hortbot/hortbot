package bnsq

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/leononame/clock"
	"github.com/nsqio/go-nsq"
	"github.com/zikaeroh/ctxlog"
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

func newSubscriber(addr string, topic string, channel string, opts ...SubscriberOption) *subscriber {
	s := &subscriber{
		addr:    addr,
		topic:   topic,
		channel: channel,
	}

	for _, opt := range opts {
		opt.applyToSubscriber(s)
	}

	if s.clk == nil {
		s.clk = clock.New()
	}

	if s.config == nil {
		s.config = defaultConfig()
	}

	return s
}

func (s *subscriber) run(ctx context.Context, fn func(m *message) error) error {
	consumer, err := nsq.NewConsumer(s.topic, s.channel, s.config)
	if err != nil {
		ctxlog.Error(ctx, "error creating consumer", zap.Error(err))
		return fmt.Errorf("creating consumer: %w", err)
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
		return fmt.Errorf("connecting to server: %w", err)
	}

	<-ctx.Done()
	return ctx.Err()
}
