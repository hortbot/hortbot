package redis

import (
	"context"
	"time"

	"github.com/opentracing/opentracing-go"
)

const authState = "auth_state"

func (db *DB) SetAuthState(ctx context.Context, state string, expiry time.Duration) error {
	span, ctx := opentracing.StartSpanFromContext(ctx, "SetAuthState")
	defer span.Finish()

	client := db.client.WithContext(ctx)
	key := buildKey(authState, state)
	return mark(client, key, expiry)
}

func (db *DB) CheckAuthState(ctx context.Context, state string) (bool, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "CheckAuthState")
	defer span.Finish()

	client := db.client.WithContext(ctx)
	key := buildKey(authState, state)
	return checkAndDelete(client, key)
}
