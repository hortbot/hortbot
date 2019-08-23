package bnsq

import (
	"context"
	"encoding/json"
	"time"

	"github.com/leononame/clock"
	"github.com/nsqio/go-nsq"
)

const (
	sendMessageTopic = "irc.send_message."
)

func newConfig() *nsq.Config {
	config := nsq.NewConfig()
	config.LookupdPollInterval = 5 * time.Second
	return config
}

type SendMessage struct {
	Timestamp time.Time
	Origin    string
	Target    string
	Message   string
}

type SendMessageProducer struct {
	addr string
	clk  clock.Clock

	ready    chan (struct{})
	producer *nsq.Producer
}

func NewSendMessageProducer(addr string, clk clock.Clock) *SendMessageProducer {
	if clk == nil {
		clk = clock.New()
	}

	return &SendMessageProducer{
		addr:  addr,
		clk:   clk,
		ready: make(chan struct{}),
	}
}

func (p *SendMessageProducer) Run(ctx context.Context) error {
	producer, err := nsq.NewProducer(p.addr, newConfig())
	if err != nil {
		return err
	}
	defer producer.Stop()

	producer.SetLogger(nsqLoggerFrom(ctx), nsq.LogLevelInfo)

	p.producer = producer
	close(p.ready)

	if err := producer.Ping(); err != nil {
		return err
	}

	<-ctx.Done()
	return ctx.Err()
}

func (p *SendMessageProducer) SendMessage(origin, target, message string) error {
	<-p.ready

	m := &SendMessage{
		Timestamp: p.clk.Now(),
		Origin:    origin,
		Target:    target,
		Message:   message,
	}

	data, err := json.Marshal(m)
	if err != nil {
		return err
	}

	return p.producer.Publish(sendMessageTopic+origin, data)
}

type SendMessageHandler func(*SendMessage)

type SendMessageConsumer struct {
	addr    string
	topic   string
	channel string
	handler SendMessageHandler
}

func NewSendMessageConsumer(addr string, origin string, channel string, handler SendMessageHandler) *SendMessageConsumer {
	return &SendMessageConsumer{
		addr:    addr,
		topic:   sendMessageTopic + origin,
		channel: channel,
		handler: handler,
	}
}

func (c *SendMessageConsumer) Run(ctx context.Context) error {
	consumer, err := nsq.NewConsumer(c.topic, c.channel, newConfig())
	if err != nil {
		return err
	}
	defer consumer.Stop()

	consumer.SetLogger(nsqLoggerFrom(ctx), nsq.LogLevelInfo)

	consumer.AddHandler(nsq.HandlerFunc(func(msg *nsq.Message) error {
		msg.Finish()

		m := &SendMessage{}

		if err := json.Unmarshal(msg.Body, m); err != nil {
			return nil
		}

		c.handler(m)
		return nil
	}))

	if err := consumer.ConnectToNSQD(c.addr); err != nil {
		return err
	}

	<-ctx.Done()
	return ctx.Err()
}
