package bot

import (
	"context"
	"math/rand"
)

//go:generate gobin -m -run github.com/maxbrunsfeld/counterfeiter/v6 -generate

//counterfeiter:generate . Sender
type Sender interface {
	SendMessage(ctx context.Context, origin, target, message string) error
}

//counterfeiter:generate . Notifier
type Notifier interface {
	NotifyChannelUpdates(ctx context.Context, botName string) error
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
