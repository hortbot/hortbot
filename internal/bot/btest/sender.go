package btest

import "context"

//go:generate go run github.com/matryer/moq -fmt goimports -out sender_mocks.go . Sender

// Sender sends a single message back via an origin (bot name) to the specified target (channel).
type Sender interface {
	SendMessage(ctx context.Context, origin, target, message string) error
}
