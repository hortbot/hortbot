package templates

import (
	"context"

	"github.com/hortbot/hortbot/internal/pkg/ctxkey"
)

var (
	brandKey = ctxkey.NewContextKey("brand", "")
	userKey  = ctxkey.NewContextKey("user", "")
)

// WithBrand returns a new context with the given brand value.
func WithBrand(ctx context.Context, brand string) context.Context {
	return brandKey.WithValue(ctx, brand)
}

// getBrand returns the brand from the context.
func getBrand(ctx context.Context) string {
	return brandKey.Value(ctx)
}

// WithUser returns a new context with the given user value.
func WithUser(ctx context.Context, user string) context.Context {
	return userKey.WithValue(ctx, user)
}

// getUser returns the user from the context.
func getUser(ctx context.Context) string {
	return userKey.Value(ctx)
}
