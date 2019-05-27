package bot

//go:generate gobin -m -run github.com/maxbrunsfeld/counterfeiter/v6 . Sender

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

//go:generate gobin -m -run github.com/maxbrunsfeld/counterfeiter/v6 . Notifier

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
