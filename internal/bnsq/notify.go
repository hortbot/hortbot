package bnsq

import (
	"context"

	"github.com/hortbot/hortbot/internal/pkg/ctxlog"
	"go.uber.org/zap"
)

const (
	notifyChannelUpdatesTopic = "notify.channel_updates."
)

// ChannelUpdatesNotification is a notification that the IRC service should
// update its channel list and potentially join/part any changed channels.
type ChannelUpdatesNotification struct {
	BotName string `json:"bot_name"`
}

// NotifyPublisher publishes notifications to NSQ.
type NotifyPublisher struct {
	publisher *publisher
}

// NewNotifyPublisher creates a new NotifyPublisher that will connect to
// the specified address.
func NewNotifyPublisher(addr string, opts ...PublisherOption) *NotifyPublisher {
	return &NotifyPublisher{
		publisher: newPublisher(addr, opts...),
	}
}

// Run runs the publisher until the context is canceled.
func (p *NotifyPublisher) Run(ctx context.Context) error {
	return p.publisher.run(ctx)
}

// NotifyChannelUpdates publishes a channel update notification, returning when
// the notification has finished publishing or the context is canceled.
func (p *NotifyPublisher) NotifyChannelUpdates(ctx context.Context, botName string) error {
	return p.publisher.publish(ctx, notifyChannelUpdatesTopic+botName, &ChannelUpdatesNotification{
		BotName: botName,
	})
}

// NotifySubscriber subscribes to notifications, executing a callback for each
// incoming notification.
type NotifySubscriber struct {
	Addr                   string
	BotName                string
	Channel                string
	Opts                   []SubscriberOption
	OnNotifyChannelUpdates func(n *ChannelUpdatesNotification, metadata *Metadata) error
}

// Run runs the subscriber until the context is canceled.
func (s *NotifySubscriber) Run(ctx context.Context) error {
	subscriber := newSubscriber(s.Addr, notifyChannelUpdatesTopic+s.BotName, s.Channel, s.Opts...)

	return subscriber.run(ctx, func(m *message) error {
		n := &ChannelUpdatesNotification{}
		if err := m.payload(n); err != nil {
			ctxlog.Warn(ctx, "error decoding ChannelUpdatesNotification", zap.Error(err))
			return nil
		}
		return s.OnNotifyChannelUpdates(n, &m.Metadata)
	})
}
