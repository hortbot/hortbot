package bnsqmeta

import (
	"context"
	"time"
)

type contextKey int

const timestampKey contextKey = iota

// Timestamp gets the bnsq metadata timestamp, or zero if not found.
func Timestamp(ctx context.Context) time.Time {
	t, _ := ctx.Value(timestampKey).(time.Time)
	return t
}

// WithTimestamp adds bnsq metadata timestamp to the context.
func WithTimestamp(ctx context.Context, t time.Time) context.Context {
	return context.WithValue(ctx, timestampKey, t)
}
