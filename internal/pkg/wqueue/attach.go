package wqueue

import (
	"context"

	"github.com/zikaeroh/ctxjoin"
)

// An Attacher attaches an extra cancelletion to the provided context.
type Attacher func(ctx context.Context) (context.Context, context.CancelFunc)

func attachFunc(ctx context.Context) Attacher {
	if ctx.Done() == nil {
		return func(ctx context.Context) (context.Context, context.CancelFunc) {
			return ctx, func() {}
		}
	}

	return func(base context.Context) (context.Context, context.CancelFunc) {
		return ctxjoin.AddCancel(base, ctx)
	}
}
