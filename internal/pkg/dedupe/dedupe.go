// Package dedupe describes a common interface for deduplicating things by a
// string ID.
package dedupe

import "context"

// Deduplicator deduplicates things by a string ID.
type Deduplicator interface {
	// Mark marks the item as seen.
	Mark(ctx context.Context, id string) error

	// Check checks if the item has been seen.
	Check(ctx context.Context, id string) (seen bool, err error)

	// CheckAndMark marks that an item has been seen, and returns true
	// if the item had been previously seen (before marked by this function).
	CheckAndMark(ctx context.Context, id string) (seen bool, err error)
}

// NeverSeen is a Deduplicator which always reports that it has not seen the ID passed to it.
var NeverSeen Deduplicator = neverSeen{}

type neverSeen struct{}

func (neverSeen) Mark(context.Context, string) error { return nil }

func (neverSeen) Check(context.Context, string) (bool, error) { return false, nil }

func (neverSeen) CheckAndMark(context.Context, string) (bool, error) { return false, nil }

//go:generate gobin -m -run github.com/maxbrunsfeld/counterfeiter/v6 . Deduplicator
