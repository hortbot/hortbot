package bot

import (
	"context"
	"math/rand"
	"sync"
	"time"
)

//go:generate go run github.com/matryer/moq -fmt goimports -out botmocks/mocks.go -pkg botmocks . Sender Notifier Rand

// Sender sends a single message back via an origin (bot name) to the specified target (channel).
type Sender interface {
	SendMessage(ctx context.Context, origin, target, message string) error
}

// Notifier sends notifications.
type Notifier interface {
	NotifyChannelUpdates(ctx context.Context, botName string) error
}

// Rand provides random number generation.
type Rand interface {
	Intn(n int) int
	Float64() float64
}

type pooledRand struct{}

var _ Rand = pooledRand{}

var randPool = sync.Pool{
	New: func() interface{} {
		source := rand.NewSource(time.Now().Unix())
		return rand.New(source) //nolint:gosec
	},
}

func (pooledRand) Intn(n int) int {
	rand := randPool.Get().(*rand.Rand)
	defer randPool.Put(rand)
	return rand.Intn(n)
}

func (pooledRand) Float64() float64 {
	rand := randPool.Get().(*rand.Rand)
	defer randPool.Put(rand)
	return rand.Float64()
}
