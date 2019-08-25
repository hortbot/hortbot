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

type SendMessage struct {
	Timestamp time.Time
	Origin    string
	Target    string
	Message   string
}

type SendMessageProducer struct {
	clk      clock.Clock
	producer *nsqProducer
}

func NewSendMessageProducer(addr string, clk clock.Clock) *SendMessageProducer {
	if clk == nil {
		clk = clock.New()
	}

	return &SendMessageProducer{
		clk:      clk,
		producer: newProducer(addr),
	}
}

func (p *SendMessageProducer) Run(ctx context.Context) error {
	return p.producer.run(ctx)
}

func (p *SendMessageProducer) SendMessage(origin, target, message string) error {
	producer := p.producer.get()

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

	return producer.Publish(sendMessageTopic+origin, data)
}

type SendMessageConsumer struct {
	Addr          string
	Origin        string
	Channel       string
	OnSendMessage func(*SendMessage)
}

func (c *SendMessageConsumer) Run(ctx context.Context) error {
	consumer, err := nsq.NewConsumer(sendMessageTopic+c.Origin, c.Channel, newConfig())
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

		c.OnSendMessage(m)
		return nil
	}))

	if err := consumer.ConnectToNSQD(c.Addr); err != nil {
		return err
	}

	<-ctx.Done()
	return ctx.Err()
}
