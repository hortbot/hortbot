package bot

import (
	"context"
	"math/rand"
	"time"

	"github.com/hortbot/hortbot/internal/pkg/pool"
)

//go:generate go run github.com/matryer/moq -fmt goimports -out botmocks/mocks.go -pkg botmocks . Rand EventsubUpdateNotifier

type Message interface {
	Origin() string
	ID() string
	Timestamp() time.Time
	BroadcasterLogin() string
	BroadcasterID() int64
	UserLogin() string
	UserDisplay() string
	UserID() int64
	Message() (message string, me bool)
	EmoteCount() int
	UserAccessLevel() AccessLevel
}

// EventsubUpdateNotifier sends notifications.
type EventsubUpdateNotifier interface {
	NotifyEventsubUpdates(ctx context.Context) error
}

// Rand provides random number generation.
type Rand interface {
	Intn(n int) int
	Float64() float64
}

type pooledRand struct{}

var _ Rand = pooledRand{}

var randPool = pool.NewPool(func() *rand.Rand {
	source := rand.NewSource(time.Now().Unix())
	return rand.New(source) //nolint:gosec
})

func (pooledRand) Intn(n int) int {
	rand := randPool.Get()
	defer randPool.Put(rand)
	return rand.Intn(n)
}

func (pooledRand) Float64() float64 {
	rand := randPool.Get()
	defer randPool.Put(rand)
	return rand.Float64()
}
