package bnsq

import (
	"time"

	"github.com/nsqio/go-nsq"
)

const (
	IncomingTopic             = incomingTopic
	NotifyChannelUpdatesTopic = notifyChannelUpdatesTopic
	SendMessageTopic          = sendMessageTopic
)

type Message = message

func TestingSleep(d time.Duration) {
	testingSleep.Store(d)
}

func DefaultConfig() *nsq.Config {
	return defaultConfig()
}
