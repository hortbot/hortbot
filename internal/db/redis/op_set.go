package redis

import (
	"context"

	"github.com/redis/go-redis/v9"
)

// setAdd adds the value to a set.
func setAdd(ctx context.Context, client redis.Cmdable, key string, value string) error {
	return client.SAdd(ctx, key, value).Err()
}

// setPop pops a value from a set at random.
func setPop(ctx context.Context, client redis.Cmdable, key string) (string, bool, error) {
	v, err := client.SPop(ctx, key).Result()

	if err == nil {
		return v, true, nil
	}

	return "", false, ignoreRedisNil(err)
}

// setPopN pops a N values from a set at random.
func setPopN(ctx context.Context, client redis.Cmdable, key string, n int64) ([]string, error) {
	return client.SPopN(ctx, key, n).Result()
}

// setLen gets the length of a set. Sets which do not exist are treated as empty.
func setLen(ctx context.Context, client redis.Cmdable, key string) (int64, error) {
	return client.SCard(ctx, key).Result()
}

// setClear clears a set from the database.
func setClear(ctx context.Context, client redis.Cmdable, key string) error {
	return client.Del(ctx, key).Err()
}
