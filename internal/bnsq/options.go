package bnsq

import (
	"time"

	"github.com/leononame/clock"
	"github.com/nsqio/go-nsq"
)

// PublisherOption is an option for a publisher.
type PublisherOption interface {
	applyToPublisher(*publisher)
}

// SubscriberOption is an option for a subscriber.
type SubscriberOption interface {
	applyToSubscriber(*subscriber)
}

// Option is an option for both a publisher and a subscriber.
type Option interface {
	PublisherOption
	SubscriberOption
}

type clockOption struct {
	clk clock.Clock
}

// WithClock sets the clock used internally. If not set or nil, a real clock
// is used.
func WithClock(clk clock.Clock) Option {
	return clockOption{clk: clk}
}

func (c clockOption) applyToPublisher(p *publisher) {
	p.clk = c.clk
}

func (c clockOption) applyToSubscriber(s *subscriber) {
	s.clk = c.clk
}

type configOption struct {
	config *nsq.Config
}

// WithConfig sets the NSQ config used for a NSQ connection. If not set or nil,
// the default NSQ configuration will be nil.
func WithConfig(config *nsq.Config) Option {
	return configOption{config: config}
}

func (c configOption) applyToPublisher(p *publisher) {
	p.config = c.config
}

func (c configOption) applyToSubscriber(s *subscriber) {
	s.config = c.config
}

type maxAgeOption struct {
	d time.Duration
}

// WithMaxAge sets the maximum age that a subscriber will accept. If a message
// is too old, it will be dropped.
func WithMaxAge(d time.Duration) SubscriberOption {
	return maxAgeOption{d: d}
}

func (c maxAgeOption) applyToSubscriber(s *subscriber) {
	s.maxAge = c.d
}
