package correlation

import (
	"context"

	"github.com/hortbot/hortbot/internal/pkg/ctxlog"
	"github.com/rs/xid"
	"go.uber.org/zap"
)

const zapName = "xid"

type contextKey int

const idKey contextKey = iota

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

	ctx = context.WithValue(ctx, idKey, id)
	return ctxlog.With(ctx, zap.String(zapName, id.String()))
}

// FromContext fetches the correlation ID from the context, or returns a nil
// ID if not found.
func FromContext(ctx context.Context) xid.ID {
	id, _ := ctx.Value(idKey).(xid.ID)
	return id
}
