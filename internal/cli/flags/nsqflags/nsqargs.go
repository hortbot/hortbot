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
