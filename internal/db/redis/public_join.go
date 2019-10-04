package redis

import (
	"context"

	"github.com/go-redis/redis/v7"
	"go.opencensus.io/trace"
)

const keyPublicJoin = keyStr("public_join")

func (db *DB) PublicJoin(ctx context.Context, botName string) (*bool, error) {
	ctx, span := trace.StartSpan(ctx, "PublicJoin")
	defer span.End()

	client := db.client.WithContext(ctx)
	key := buildKey(keyPublicJoin.is(botName))

	v, err := client.Get(key).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, nil
		}
		return nil, err
	}

	b := v == "1"
	return &b, nil
}

func (db *DB) SetPublicJoin(ctx context.Context, botName string, enable bool) error {
	ctx, span := trace.StartSpan(ctx, "SetPublicJoin")
	defer span.End()

	client := db.client.WithContext(ctx)
	key := buildKey(keyPublicJoin.is(botName))

	if enable {
		return client.Set(key, "1", 0).Err()
	}

	return client.Set(key, "0", 0).Err()
}

func (db *DB) UnsetPublicJoin(ctx context.Context, botName string) error {
	ctx, span := trace.StartSpan(ctx, "UnsetPublicJoin")
	defer span.End()

	client := db.client.WithContext(ctx)
	key := buildKey(keyPublicJoin.is(botName))

	return client.Del(key).Err()
}
