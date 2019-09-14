package redis

import (
	"context"
	"time"

	"go.opencensus.io/trace"
)

const authState = "auth_state"

func (db *DB) SetAuthState(ctx context.Context, state string, expiry time.Duration) error {
	ctx, span := trace.StartSpan(ctx, "SetAuthState")
	defer span.End()

	client := db.client.WithContext(ctx)
	key := buildKey(authState, state)
	return mark(client, key, expiry)
}

func (db *DB) CheckAuthState(ctx context.Context, state string) (bool, error) {
	ctx, span := trace.StartSpan(ctx, "CheckAuthState")
	defer span.End()

	client := db.client.WithContext(ctx)
	key := buildKey(authState, state)
	return checkAndDelete(client, key)
}
