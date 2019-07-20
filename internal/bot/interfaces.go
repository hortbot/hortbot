package bot

import "math/rand"

//go:generate gobin -m -run github.com/maxbrunsfeld/counterfeiter/v6 -generate

//counterfeiter:generate . Sender
type Sender interface {
	SendMessage(origin, target, message string) error
}

type SenderFuncs struct {
	SendMessageFunc func(origin, target, message string) error
}

var _ Sender = SenderFuncs{}

func (s SenderFuncs) SendMessage(origin, target, message string) error {
	return s.SendMessageFunc(origin, target, message)
}

//counterfeiter:generate . Notifier
type Notifier interface {
	NotifyChannelUpdates(botName string)
}

type NotifierFuncs struct {
	NotifyChannelUpdatesFunc func(botName string)
}

var _ Notifier = NotifierFuncs{}

func (n NotifierFuncs) NotifyChannelUpdates(botName string) {
	n.NotifyChannelUpdatesFunc(botName)
}

//counterfeiter:generate . Rand
type Rand interface {
	Intn(n int) int
	Float64() float64
}

type globalRand struct{}

var _ Rand = globalRand{}

func (globalRand) Intn(n int) int {
	return rand.Intn(n)
}

func (globalRand) Float64() float64 {
	return rand.Float64()
}
