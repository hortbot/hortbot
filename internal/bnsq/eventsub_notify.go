package bnsq

import (
	"context"

	"github.com/zikaeroh/ctxlog"
	"go.uber.org/zap"
)

const (
	EventsubNotifyTopic = "eventsub.notify"
)

// EventsubNotify is an EventsubNotify message as sent over NSQ.
type EventsubNotify struct{}

// EventsubNotifyPublisher publishes EventsubNotify messages to NSQ.
type EventsubNotifyPublisher struct {
	publisher *publisher
}

// NewEventsubNotifyPublisher creates a new EventsubNotifyPublisher that will connect to
// the specified address.
func NewEventsubNotifyPublisher(addr string, opts ...PublisherOption) *EventsubNotifyPublisher {
	return &EventsubNotifyPublisher{
		publisher: newPublisher(addr, opts...),
	}
}

// Run runs the publisher until the context is canceled.
func (p *EventsubNotifyPublisher) Run(ctx context.Context) error {
	return p.publisher.run(ctx)
}

// Publish publishes an EventsubNotify message, returning when the message has finished
// publishing or the context is canceled.
func (p *EventsubNotifyPublisher) NotifyEventsubUpdates(ctx context.Context) error {
	return p.publisher.publish(ctx, EventsubNotifyTopic, &EventsubNotify{})
}

// EventsubNotifySubscriber subscribes to EventsubNotify messages, executing OnEventsubNotify for
// each message.
type EventsubNotifySubscriber struct {
	Addr             string
	Channel          string
	Opts             []SubscriberOption
	OnEventsubNotify func(i *EventsubNotify, metadata *Metadata) error
}

// Run runs the subscriber until the context is canceled.
func (s *EventsubNotifySubscriber) Run(ctx context.Context) error {
	subscriber := newSubscriber(s.Addr, EventsubNotifyTopic, s.Channel, s.Opts...)

	return subscriber.run(ctx, func(m *message) error {
		i := &EventsubNotify{}
		if err := m.payload(i); err != nil {
			ctxlog.Warn(ctx, "error decoding EventsubNotify", zap.Error(err))
			return nil
		}
		return s.OnEventsubNotify(i, &m.Metadata)
	})
}
