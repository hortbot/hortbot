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

	client := db.client.WithContext(ctx)
	rkey := buildKey(keyAuthState.is(key))
	return client.Set(rkey, b, expiry).Err()
}

// GetAuthState gets the arbitrary authentication state for the login workflow.
func (db *DB) GetAuthState(ctx context.Context, key string, v interface{}) (bool, error) {
	ctx, span := trace.StartSpan(ctx, "GetAuthState")
	defer span.End()

	client := db.client.WithContext(ctx)
	rkey := buildKey(keyAuthState.is(key))

	pipeline := client.TxPipeline()
	get := pipeline.Get(rkey)
	pipeline.Del(rkey)
	_, _ = pipeline.Exec() // Error is propogated below.

	b, err := get.Bytes()
	if err != nil {
		return false, ignoreRedisNil(err)
	}

	return true, json.Unmarshal(b, v)
}
