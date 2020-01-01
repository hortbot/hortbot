package bnsq

import (
	"context"

	"github.com/hortbot/hortbot/internal/pkg/ctxlog"
	"github.com/jakebailey/irc"
	"go.uber.org/zap"
)

const (
	incomingTopic = "irc.incoming"
)

type Incoming struct {
	Origin  string       `json:"origin"`
	Message *irc.Message `json:"message"`
}

type IncomingPublisher struct {
	publisher *publisher
}

func NewIncomingPublisher(addr string, opts ...PublisherOption) *IncomingPublisher {
	return &IncomingPublisher{
		publisher: newPublisher(addr, opts...),
	}
}

func (p *IncomingPublisher) Run(ctx context.Context) error {
	return p.publisher.run(ctx)
}

func (p *IncomingPublisher) Publish(ctx context.Context, origin string, m *irc.Message) error {
	return p.publisher.publish(ctx, incomingTopic, &Incoming{
		Origin:  origin,
		Message: m,
	})
}

type IncomingSubscriber struct {
	Addr       string
	Channel    string
	Opts       []SubscriberOption
	OnIncoming func(i *Incoming, metadata *Metadata) error
}

func (s *IncomingSubscriber) Run(ctx context.Context) error {
	subscriber := newSubscriber(s.Addr, incomingTopic, s.Channel, s.Opts...)

	return subscriber.run(ctx, func(m *message) error {
		i := &Incoming{}
		if err := m.payload(i); err != nil {
			ctxlog.Warn(ctx, "error decoding Incoming", zap.Error(err))
			return nil
		}
		return s.OnIncoming(i, &m.Metadata)
	})
}
