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

// Incoming is an incoming IRC message as sent over NSQ>
type Incoming struct {
	Origin  string       `json:"origin"`
	Message *irc.Message `json:"message"`
}

// IncomingPublisher publishes incoming messages to NSQ.
type IncomingPublisher struct {
	publisher *publisher
}

// NewIncomingPublisher creates a new IncomingPublisher that will connect to
// the specified address.
func NewIncomingPublisher(addr string, opts ...PublisherOption) *IncomingPublisher {
	return &IncomingPublisher{
		publisher: newPublisher(addr, opts...),
	}
}

// Run runs the publisher until the context is canceled.
func (p *IncomingPublisher) Run(ctx context.Context) error {
	return p.publisher.run(ctx)
}

// Publish publishes an incoming message, returning when the message has finished
// publishing or the context is canceled.
func (p *IncomingPublisher) Publish(ctx context.Context, origin string, m *irc.Message) error {
	return p.publisher.publish(ctx, incomingTopic, &Incoming{
		Origin:  origin,
		Message: m,
	})
}

// IncomingSubscriber subscribes to incoming messages, executing OnIncoming for
// each message.
type IncomingSubscriber struct {
	Addr       string
	Channel    string
	Opts       []SubscriberOption
	OnIncoming func(i *Incoming, metadata *Metadata) error
}

// Run runs the subscriber until the context is canceled.
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
