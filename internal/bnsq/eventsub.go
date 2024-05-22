package bnsq

import (
	"context"

	"github.com/hortbot/hortbot/internal/pkg/apiclient/twitch/eventsub"
	"github.com/zikaeroh/ctxlog"
	"go.uber.org/zap"
)

const (
	IncomingWebsocketMessageTopic = "eventsub.incoming"
)

// IncomingWebsocketMessage is an IncomingWebsocketMessage IRC message as sent over NSQ.
type IncomingWebsocketMessage struct {
	Message *eventsub.WebsocketMessage `json:"message"`
}

// IncomingWebsocketMessagePublisher publishes IncomingWebsocketMessage messages to NSQ.
type IncomingWebsocketMessagePublisher struct {
	publisher *publisher
}

// NewIncomingWebsocketMessagePublisher creates a new IncomingWebsocketMessagePublisher that will connect to
// the specified address.
func NewIncomingWebsocketMessagePublisher(addr string, opts ...PublisherOption) *IncomingWebsocketMessagePublisher {
	return &IncomingWebsocketMessagePublisher{
		publisher: newPublisher(addr, opts...),
	}
}

// Run runs the publisher until the context is canceled.
func (p *IncomingWebsocketMessagePublisher) Run(ctx context.Context) error {
	return p.publisher.run(ctx)
}

// Publish publishes an IncomingWebsocketMessage message, returning when the message has finished
// publishing or the context is canceled.
func (p *IncomingWebsocketMessagePublisher) Publish(ctx context.Context, m *eventsub.WebsocketMessage) error {
	return p.publisher.publish(ctx, IncomingWebsocketMessageTopic, &IncomingWebsocketMessage{
		Message: m,
	})
}

// IncomingWebsocketMessageSubscriber subscribes to IncomingWebsocketMessage messages, executing OnIncomingWebsocketMessage for
// each message.
type IncomingWebsocketMessageSubscriber struct {
	Addr                       string
	Channel                    string
	Opts                       []SubscriberOption
	OnIncomingWebsocketMessage func(i *IncomingWebsocketMessage, metadata *Metadata) error
}

// Run runs the subscriber until the context is canceled.
func (s *IncomingWebsocketMessageSubscriber) Run(ctx context.Context) error {
	subscriber := newSubscriber(s.Addr, IncomingWebsocketMessageTopic, s.Channel, s.Opts...)

	return subscriber.run(ctx, func(m *message) error {
		i := &IncomingWebsocketMessage{}
		if err := m.payload(i); err != nil {
			ctxlog.Warn(ctx, "error decoding IncomingWebsocketMessage", zap.Error(err))
			return nil
		}
		return s.OnIncomingWebsocketMessage(i, &m.Metadata)
	})
}
