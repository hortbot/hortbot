package bot

import (
	"context"
	"encoding/json"
	"math/rand/v2"
	"time"
)

//go:generate go run github.com/matryer/moq -fmt goimports -out botmocks/mocks.go -pkg botmocks . Rand EventsubUpdateNotifier

type Message interface {
	json.Marshaler
	Origin() string
	ID() string
	Timestamp() time.Time
	BroadcasterLogin() string
	BroadcasterDisplay() string
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

type defaultRand struct{}

var _ Rand = defaultRand{}

func (defaultRand) Intn(n int) int {
	return rand.N(n) //nolint:gosec
}

func (defaultRand) Float64() float64 {
	return rand.Float64() //nolint:gosec
}
