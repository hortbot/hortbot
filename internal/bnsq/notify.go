package bnsq

import "context"

const (
	notifyChannelUpdatesTopic = "notify.channel_updates."
)

type ChannelUpdatesNotification struct {
	BotName string `json:"bot_name"`
}

type NotifyPublisher struct {
	publisher *publisher
}

func NewNotifyPublisher(addr string, opts ...PublisherOption) *NotifyPublisher {
	return &NotifyPublisher{
		publisher: newPublisher(addr, opts...),
	}
}

func (p *NotifyPublisher) Run(ctx context.Context) error {
	return p.publisher.run(ctx)
}

func (p *NotifyPublisher) NotifyChannelUpdates(ctx context.Context, botName string) error {
	return p.publisher.publish(ctx, notifyChannelUpdatesTopic+botName, &ChannelUpdatesNotification{
		BotName: botName,
	})
}

type NotifySubscriber struct {
	Addr                   string
	BotName                string
	Channel                string
	Opts                   []SubscriberOption
	OnNotifyChannelUpdates func(*ChannelUpdatesNotification)
}

func (s *NotifySubscriber) Run(ctx context.Context) error {
	subscriber := newSubscriber(s.Addr, notifyChannelUpdatesTopic+s.BotName, s.Channel, s.Opts...)

	return subscriber.run(ctx, func(m *message) {
		n := &ChannelUpdatesNotification{}
		if err := m.payload(n); err != nil {
			return
		}
		s.OnNotifyChannelUpdates(n)
	})
}
