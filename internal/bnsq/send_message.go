package bnsq

import (
	"context"

	"github.com/hortbot/hortbot/internal/pkg/ctxlog"
	"go.uber.org/zap"
)

const (
	sendMessageTopic = "irc.send_message."
)

// SendMessage is an outgoing message as sent over NSQ.
type SendMessage struct {
	Origin  string `json:"origin"`
	Target  string `json:"target"`
	Message string `json:"message"`
}

// SendMessagePublisher publishes outgoing messages to NSQ.
type SendMessagePublisher struct {
	publisher *publisher
}

// NewSendMessagePublisher creates a new SendMessagePublisher that will connect to
// the specified address.
func NewSendMessagePublisher(addr string, opts ...PublisherOption) *SendMessagePublisher {
	return &SendMessagePublisher{
		publisher: newPublisher(addr, opts...),
	}
}

// Run runs the publisher until the context is canceled.
func (p *SendMessagePublisher) Run(ctx context.Context) error {
	return p.publisher.run(ctx)
}

// SendMessage publishes a outgoing message, returning when the message
// has finished publishing or the context is canceled.
func (p *SendMessagePublisher) SendMessage(ctx context.Context, origin, target, message string) error {
	return p.publisher.publish(ctx, sendMessageTopic+origin, &SendMessage{
		Origin:  origin,
		Target:  target,
		Message: message,
	})
}

// SendMessageSubscriber subscribes to outgoing message, executing OnSendMessage
// for each.
type SendMessageSubscriber struct {
	Addr          string
	Origin        string
	Channel       string
	Opts          []SubscriberOption
	OnSendMessage func(sm *SendMessage, metadata *Metadata) error
}

// Run runs the subscriber until the context is canceled.
func (s *SendMessageSubscriber) Run(ctx context.Context) error {
	subscriber := newSubscriber(s.Addr, sendMessageTopic+s.Origin, s.Channel, s.Opts...)

	return subscriber.run(ctx, func(m *message) error {
		sm := &SendMessage{}
		if err := m.payload(sm); err != nil {
			ctxlog.Warn(ctx, "error decoding SendMessage", zap.Error(err))
			return nil
		}
		return s.OnSendMessage(sm, &m.Metadata)
	})
}
