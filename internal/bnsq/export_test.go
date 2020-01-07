package bnsq

const (
	IncomingTopic             = incomingTopic
	NotifyChannelUpdatesTopic = notifyChannelUpdatesTopic
	SendMessageTopic          = sendMessageTopic
)

type Message = message

var (
	TestingSleep  = testingSleep.Store
	DefaultConfig = defaultConfig
	NsqLoggerFrom = nsqLoggerFrom
)
