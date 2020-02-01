package bot

import (
	"context"
	"math/rand"
)

//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6 -generate

//counterfeiter:generate . Sender

// Sender sends a single message back via an origin (bot name) to the specified target (channel).
type Sender interface {
	SendMessage(ctx context.Context, origin, target, message string) error
}

//counterfeiter:generate . Notifier

// Notifier sends notifications.
type Notifier interface {
	NotifyChannelUpdates(ctx context.Context, botName string) error
}

//counterfeiter:generate . Rand

// Rand provides random number generation.
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
