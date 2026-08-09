// Package contextx provides helpers for context lifecycles.
package contextx

import (
	"context"
	"time"
)

// WithGracePeriod returns a context that remains active for grace after parent
// is canceled. Calling cancel ends it immediately.
func WithGracePeriod(parent context.Context, grace time.Duration) (context.Context, context.CancelFunc) {
	if grace <= 0 {
		panic("non-positive grace period")
	}

	ctx, cancel := context.WithCancel(context.WithoutCancel(parent))
	go func() {
		select {
		case <-parent.Done():
		case <-ctx.Done():
			return
		}

		timer := time.NewTimer(grace)
		defer timer.Stop()
		select {
		case <-timer.C:
			cancel()
		case <-ctx.Done():
		}
	}()
	return ctx, cancel
}
