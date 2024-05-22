// Package nsqflags processes NSQ-related flags.
package nsqflags

import (
	"time"

	"github.com/hortbot/hortbot/internal/bnsq"
)

// NSQ contains NSQ flags.
type NSQ struct {
	Addr    string `long:"nsq-addr" env:"HB_NSQ_ADDR" description:"NSQD address" required:"true"`
	Channel string `long:"nsq-channel" env:"HB_NSQ_CHANNEL" description:"NSQ subscription channel"`
}

// Default contains the default flags. Make a copy of this, do not reuse.
var Default = NSQ{
	Channel: "queue",
}

// NewIncomingPublisher creates a new IncomingPublisher.
func (args *NSQ) NewIncomingPublisher() *bnsq.IncomingPublisher {
	return bnsq.NewIncomingPublisher(args.Addr)
}

// NewIncomingSubscriber creates a new IncomingSubscriber.
func (args *NSQ) NewIncomingSubscriber(maxAge time.Duration, fn func(*bnsq.Incoming, *bnsq.Metadata) error) *bnsq.IncomingSubscriber {
	return &bnsq.IncomingSubscriber{
		Addr:    args.Addr,
		Channel: args.Channel,
		Opts: []bnsq.SubscriberOption{
			bnsq.WithMaxAge(maxAge),
		},
		OnIncoming: fn,
	}
}

// NewNotifyPublisher creates a new NotifyPublisher.
func (args *NSQ) NewNotifyPublisher() *bnsq.NotifyPublisher {
	return bnsq.NewNotifyPublisher(args.Addr)
}

// NewNotifySubscriber creates a new NotifySubscriber.
func (args *NSQ) NewNotifySubscriber(botName string, maxAge time.Duration, fn func(*bnsq.ChannelUpdatesNotification, *bnsq.Metadata) error) *bnsq.NotifySubscriber {
	return &bnsq.NotifySubscriber{
		Addr:    args.Addr,
		BotName: botName,
		Channel: args.Channel,
		Opts: []bnsq.SubscriberOption{
			bnsq.WithMaxAge(maxAge),
		},
		OnNotifyChannelUpdates: fn,
	}
}

// NewIncomingWebsocketMessagePublisher creates a new IncomingWebsocketMessagePublisher.
func (args *NSQ) NewIncomingWebsocketMessagePublisher() *bnsq.IncomingWebsocketMessagePublisher {
	return bnsq.NewIncomingWebsocketMessagePublisher(args.Addr)
}

// NewIncomingWebsocketMessageSubscriber creates a new IncomingWebsocketMessageSubscriber.
func (args *NSQ) NewIncomingWebsocketMessageSubscriber(maxAge time.Duration, fn func(*bnsq.IncomingWebsocketMessage, *bnsq.Metadata) error) *bnsq.IncomingWebsocketMessageSubscriber {
	return &bnsq.IncomingWebsocketMessageSubscriber{
		Addr:    args.Addr,
		Channel: args.Channel,
		Opts: []bnsq.SubscriberOption{
			bnsq.WithMaxAge(maxAge),
		},
		OnIncomingWebsocketMessage: fn,
	}
}

// NewEventsubNotifyPublisher creates a new EventsubNotifyPublisher.
func (args *NSQ) NewEventsubNotifyPublisher() *bnsq.EventsubNotifyPublisher {
	return bnsq.NewEventsubNotifyPublisher(args.Addr)
}

// NewEventsubNotifySubscriber creates a new EventsubNotifySubscriber.
func (args *NSQ) NewEventsubNotifySubscriber(maxAge time.Duration, fn func(*bnsq.EventsubNotify, *bnsq.Metadata) error) *bnsq.EventsubNotifySubscriber {
	return &bnsq.EventsubNotifySubscriber{
		Addr:    args.Addr,
		Channel: args.Channel,
		Opts: []bnsq.SubscriberOption{
			bnsq.WithMaxAge(maxAge),
		},
		OnEventsubNotify: fn,
	}
}
