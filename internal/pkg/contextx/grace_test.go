package contextx_test

import (
	"context"
	"testing"
	"time"

	"github.com/hortbot/hortbot/internal/pkg/contextx"
	"gotest.tools/v3/assert"
)

func TestWithGracePeriod(t *testing.T) {
	t.Parallel()

	parent, cancelParent := context.WithCancel(t.Context())
	ctx, cancel := contextx.WithGracePeriod(parent, 20*time.Millisecond)
	defer cancel()

	cancelParent()
	select {
	case <-ctx.Done():
		t.Fatal("grace context canceled with its parent")
	default:
	}

	select {
	case <-ctx.Done():
	case <-time.After(time.Second):
		t.Fatal("grace context did not expire")
	}
}

func TestWithGracePeriodCanBeCanceledImmediately(t *testing.T) {
	t.Parallel()

	ctx, cancel := contextx.WithGracePeriod(t.Context(), time.Hour)
	cancel()

	assert.ErrorIs(t, ctx.Err(), context.Canceled)
}
