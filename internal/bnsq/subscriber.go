package bnsq

import (
	"context"
	"encoding/json"
	"time"

	"github.com/leononame/clock"
	"github.com/nsqio/go-nsq"
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

func (s *subscriber) run(ctx context.Context, fn func(*message)) error {
	consumer, err := nsq.NewConsumer(s.topic, s.channel, s.config)
	if err != nil {
		return err
	}
	defer consumer.Stop()

	consumer.SetLogger(nsqLoggerFrom(ctx), nsq.LogLevelInfo)

	consumer.AddHandler(nsq.HandlerFunc(func(msg *nsq.Message) error {
		msg.Finish()

		if testingSleep != 0 {
			s.clk.Sleep(testingSleep)
		}

		m := &message{}

		if err := json.Unmarshal(msg.Body, m); err != nil {
			return nil
		}

		if s.maxAge > 0 && s.clk.Since(m.Timestamp) > s.maxAge {
			return nil
		}

		fn(m)
		return nil
	}))

	if err := consumer.ConnectToNSQD(s.addr); err != nil {
		return err
	}

	<-ctx.Done()
	return ctx.Err()
}
