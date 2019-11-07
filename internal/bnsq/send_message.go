package bnsq

import (
	"context"
)

const (
	sendMessageTopic = "irc.send_message."
)

type SendMessage struct {
	Origin  string `json:"origin"`
	Target  string `json:"target"`
	Message string `json:"message"`
}

type SendMessagePublisher struct {
	publisher *publisher
}

func NewSendMessagePublisher(addr string, opts ...PublisherOption) *SendMessagePublisher {
	return &SendMessagePublisher{
		publisher: newPublisher(addr, opts...),
	}
}

func (p *SendMessagePublisher) Run(ctx context.Context) error {
	return p.publisher.run(ctx)
}

func (p *SendMessagePublisher) SendMessage(ctx context.Context, origin, target, message string) error {
	return p.publisher.publish(ctx, sendMessageTopic+origin, &SendMessage{
		Origin:  origin,
		Target:  target,
		Message: message,
	})
}

type SendMessageSubscriber struct {
	Addr          string
	Origin        string
	Channel       string
	Opts          []SubscriberOption
	OnSendMessage func(sm *SendMessage, metadata *Metadata) error
}

func (s *SendMessageSubscriber) Run(ctx context.Context) error {
	subscriber := newSubscriber(s.Addr, sendMessageTopic+s.Origin, s.Channel, s.Opts...)

	return subscriber.run(ctx, func(m *message) error {
		sm := &SendMessage{}
		if err := m.payload(sm); err != nil {
			return err
		}
		return s.OnSendMessage(sm, &m.Metadata)
	})
}
