package redis

import (
	"context"
	"encoding/json"
	"time"

	"github.com/go-redis/redis/v7"
	"go.opencensus.io/trace"
)

const keyAuthState = keyStr("auth_state")

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

func (db *DB) GetAuthState(ctx context.Context, key string, v interface{}) (bool, error) {
	ctx, span := trace.StartSpan(ctx, "GetAuthState")
	defer span.End()

	client := db.client.WithContext(ctx)
	rkey := buildKey(keyAuthState.is(key))

	pipeline := client.TxPipeline()
	get := pipeline.Get(rkey)
	pipeline.Del(rkey)
	if _, err := pipeline.Exec(); err != nil {
		if err == redis.Nil {
			return false, nil
		}
		return false, err
	}

	b, err := get.Bytes()
	if err != nil {
		return false, err
	}

	return true, json.Unmarshal(b, v)
}
