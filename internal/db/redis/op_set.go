package redis

import "github.com/go-redis/redis/v7"

// setAdd adds the value to a set.
func setAdd(client redis.Cmdable, key string, value string) error {
	return client.SAdd(key, value).Err()
}

// setPop pops a value from a set at random.
func setPop(client redis.Cmdable, key string) (string, bool, error) {
	v, err := client.SPop(key).Result()

	if err == nil {
		return v, true, nil
	}

	return "", false, ignoreRedisNil(err)
}

// setLen gets the length of a set. Sets which do not exist are treated as empty.
func setLen(client redis.Cmdable, key string) (int64, error) {
	return client.SCard(key).Result()
}

// setClear clears a set from the database.
func setClear(client redis.Cmdable, key string) error {
	return client.Del(key).Err()
}
