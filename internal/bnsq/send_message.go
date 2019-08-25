package bnsq

import (
	"context"
)

const (
	sendMessageTopic = "irc.send_message."
)

type SendMessage struct {
	Origin  string
	Target  string
	Message string
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

func (p *SendMessagePublisher) SendMessage(origin, target, message string) error {
	m := &SendMessage{
		Origin:  origin,
		Target:  target,
		Message: message,
	}

	return p.publisher.publish(sendMessageTopic+origin, m)
}

type SendMessageSubscriber struct {
	Addr          string
	Origin        string
	Channel       string
	Opts          []SubscriberOption
	OnSendMessage func(*SendMessage)
}

func (s *SendMessageSubscriber) Run(ctx context.Context) error {
	subscriber := newSubscriber(s.Addr, sendMessageTopic+s.Origin, s.Channel, s.Opts...)

	return subscriber.run(ctx, func(m *message) {
		sm := &SendMessage{}
		if err := m.payload(sm); err != nil {
			return
		}
		s.OnSendMessage(sm)
	})
}
