package redis

import (
	"context"
	"encoding/json"
	"time"

	"go.opencensus.io/trace"
)

const keyAuthState = keyStr("auth_state")

// SetAuthState sets an arbitrary authentication state for the login workflow.
func (db *DB) SetAuthState(ctx context.Context, key string, value interface{}, expiry time.Duration) error {
	ctx, span := trace.StartSpan(ctx, "SetAuthState")
	defer span.End()

	b, err := json.Marshal(value)
	if err != nil {
		return err
	}

	rkey := buildKey(keyAuthState.is(key))
	return db.client.Set(ctx, rkey, b, expiry).Err()
}

// GetAuthState gets the arbitrary authentication state for the login workflow.
func (db *DB) GetAuthState(ctx context.Context, key string, v interface{}) (bool, error) {
	ctx, span := trace.StartSpan(ctx, "GetAuthState")
	defer span.End()

	rkey := buildKey(keyAuthState.is(key))

	pipeline := db.client.TxPipeline()
	get := pipeline.Get(ctx, rkey)
	pipeline.Del(ctx, rkey)
	_, _ = pipeline.Exec(ctx) // Error is propogated below.

	b, err := get.Bytes()
	if err != nil {
		return false, ignoreRedisNil(err)
	}

	return true, json.Unmarshal(b, v)
}
