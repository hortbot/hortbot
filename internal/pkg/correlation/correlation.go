// Package correlation provides correlation semantics for use across service boundaries.
package correlation

import (
	"context"

	"github.com/hortbot/hortbot/internal/pkg/ctxkey"
	"github.com/rs/xid"
	"github.com/zikaeroh/ctxlog"
	"go.uber.org/zap"
)

const zapName = "xid"

var contextKey = ctxkey.NewContextKey("correlation", xid.NilID())

// With adds a new correlation ID to the context if one has not already been
// added.
func With(ctx context.Context) context.Context {
	if !FromContext(ctx).IsNil() {
		return ctx
	}
	return WithID(ctx, xid.New())
}

// WithID adds the specified correlation ID to the context unconditionally,
// overwriting any correlation ID already present. If the given ID is nil,
// then a new ID will be used.
func WithID(ctx context.Context, id xid.ID) context.Context {
	if id.IsNil() {
		id = xid.New()
	}

	ctx = contextKey.WithValue(ctx, id)
	return ctxlog.With(ctx, zap.String(zapName, id.String()))
}

// FromContext fetches the correlation ID from the context, or returns a nil
// ID if not found.
func FromContext(ctx context.Context) xid.ID {
	return contextKey.Value(ctx)
}
