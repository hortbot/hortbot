// Package nsqflags processes NSQ-related flags.
package nsqflags

import (
	"time"

	"github.com/hortbot/hortbot/internal/bnsq"
)

type NSQ struct {
	Addr    string `long:"nsq-addr" env:"HB_NSQ_ADDR" description:"NSQD address" required:"true"`
	Channel string `long:"nsq-channel" env:"HB_NSQ_CHANNEL" description:"NSQ subscription channel"`
}

var DefaultNSQ = NSQ{
	Channel: "queue",
}

func (args *NSQ) NewIncomingPublisher() *bnsq.IncomingPublisher {
	return bnsq.NewIncomingPublisher(args.Addr)
}

func (args *NSQ) NewIncomingSubscriber(maxAge time.Duration, fn func(*bnsq.Incoming, *bnsq.Metadata) error) *bnsq.IncomingSubscriber {
	return &bnsq.IncomingSubscriber{
		Addr:    args.Addr,
		Channel: args.Channel,
		Opts: []bnsq.SubscriberOption{
			bnsq.SubscriberMaxAge(maxAge),
		},
		OnIncoming: fn,
	}
}

func (args *NSQ) NewSendMessagePublisher() *bnsq.SendMessagePublisher {
	return bnsq.NewSendMessagePublisher(args.Addr)
}

func (args *NSQ) NewSendMessageSubscriber(origin string, maxAge time.Duration, fn func(*bnsq.SendMessage, *bnsq.Metadata) error) *bnsq.SendMessageSubscriber {
	return &bnsq.SendMessageSubscriber{
		Addr:    args.Addr,
		Origin:  origin,
		Channel: args.Channel,
		Opts: []bnsq.SubscriberOption{
			bnsq.SubscriberMaxAge(maxAge),
		},
		OnSendMessage: fn,
	}
}

func (args *NSQ) NewNotifyPublisher() *bnsq.NotifyPublisher {
	return bnsq.NewNotifyPublisher(args.Addr)
}

func (args *NSQ) NewNotifySubscriber(botName string, maxAge time.Duration, fn func(*bnsq.ChannelUpdatesNotification, *bnsq.Metadata) error) *bnsq.NotifySubscriber {
	return &bnsq.NotifySubscriber{
		Addr:    args.Addr,
		BotName: botName,
		Channel: args.Channel,
		Opts: []bnsq.SubscriberOption{
			bnsq.SubscriberMaxAge(maxAge),
		},
		OnNotifyChannelUpdates: fn,
	}
}
