// Package nsqargs processes NSQ arguments.
package nsqargs

import (
	"time"

	"github.com/hortbot/hortbot/internal/bnsq"
	"go.opencensus.io/trace"
)

type NSQ struct {
	NSQAddr    string `long:"nsq-addr" env:"HB_NSQ_ADDR" description:"NSQD address" required:"true"`
	NSQChannel string `long:"nsq-channel" env:"HB_NSQ_CHANNEL" description:"NSQ subscription channel"`
}

var DefaultNSQ = NSQ{
	NSQChannel: "queue",
}

func (args *NSQ) NewIncomingPublisher() *bnsq.IncomingPublisher {
	return bnsq.NewIncomingPublisher(args.NSQAddr)
}

func (args *NSQ) NewIncomingSubscriber(maxAge time.Duration, fn func(i *bnsq.Incoming, parent trace.SpanContext) error) *bnsq.IncomingSubscriber {
	return &bnsq.IncomingSubscriber{
		Addr:    args.NSQAddr,
		Channel: args.NSQChannel,
		Opts: []bnsq.SubscriberOption{
			bnsq.SubscriberMaxAge(maxAge),
		},
		OnIncoming: fn,
	}
}

func (args *NSQ) NewSendMessagePublisher() *bnsq.SendMessagePublisher {
	return bnsq.NewSendMessagePublisher(args.NSQAddr)
}

func (args *NSQ) NewSendMessageSubscriber(origin string, maxAge time.Duration, fn func(m *bnsq.SendMessage, parent trace.SpanContext) error) *bnsq.SendMessageSubscriber {
	return &bnsq.SendMessageSubscriber{
		Addr:    args.NSQAddr,
		Origin:  origin,
		Channel: args.NSQChannel,
		Opts: []bnsq.SubscriberOption{
			bnsq.SubscriberMaxAge(maxAge),
		},
		OnSendMessage: fn,
	}
}

func (args *NSQ) NewNotifyPublisher() *bnsq.NotifyPublisher {
	return bnsq.NewNotifyPublisher(args.NSQAddr)
}

func (args *NSQ) NewNotifySubscriber(botName string, maxAge time.Duration, fn func(n *bnsq.ChannelUpdatesNotification, parent trace.SpanContext) error) *bnsq.NotifySubscriber {
	return &bnsq.NotifySubscriber{
		Addr:    args.NSQAddr,
		BotName: botName,
		Channel: args.NSQChannel,
		Opts: []bnsq.SubscriberOption{
			bnsq.SubscriberMaxAge(maxAge),
		},
		OnNotifyChannelUpdates: fn,
	}
}
