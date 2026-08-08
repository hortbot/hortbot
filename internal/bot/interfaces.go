package bot

import (
	"context"
	"encoding/json"
	"math/rand/v2"
	"time"
)

//go:generate go tool github.com/matryer/moq -fmt goimports -out botmocks/mocks.go -pkg botmocks . Rand EventsubUpdateNotifier

type ChatIdentity struct {
	ID          int64
	Login       string
	DisplayName string
}

type Message interface {
	json.Marshaler
	Bot() string
	MessageID() string
	MessageTimestamp() time.Time
	Broadcaster() ChatIdentity
	Chatter() ChatIdentity
	Text() string
	IsAction() bool
	CountEmotes() int
	ChatterAccessLevel() AccessLevel
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
