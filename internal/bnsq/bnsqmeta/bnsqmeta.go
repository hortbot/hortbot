// Package bnsqmeta propagates NSQ metadata via contexts.
//
// This package is separated from bnsq to prevent other packages from depending
// on the NSQ client libraries.
package bnsqmeta

import (
	"context"
	"time"

	"github.com/hortbot/hortbot/internal/pkg/ctxkey"
)

var timestampKey = ctxkey.NewContextKey("bnsqmeta.timestamp", time.Time{})

// Timestamp gets the bnsq metadata timestamp, or zero if not found.
func Timestamp(ctx context.Context) time.Time {
	return timestampKey.Value(ctx)
}

// WithTimestamp adds bnsq metadata timestamp to the context.
func WithTimestamp(ctx context.Context, t time.Time) context.Context {
	return timestampKey.WithValue(ctx, t)
}
