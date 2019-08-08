package rdb

import "github.com/go-redis/redis/v7"

// SetAdd adds the value to a set.
func (d *DB) SetAdd(value string, key string, more ...string) error {
	k := d.buildKey(key, more...)
	return d.client.SAdd(k, value).Err()
}

// SetPop pops a value from a set at random.
func (d *DB) SetPop(key string, more ...string) (string, bool, error) {
	k := d.buildKey(key, more...)
	v, err := d.client.SPop(k).Result()

	if err == nil {
		return v, true, nil
	}

	if err == redis.Nil {
		err = nil
	}

	return "", false, err
}

// SetLen gets the length of a set. Sets which do not exist are treated as empty.
func (d *DB) SetLen(key string, more ...string) (int64, error) {
	k := d.buildKey(key, more...)
	return d.client.SCard(k).Result()
}

// SetClear clears a set from the database.
func (d *DB) SetClear(key string, more ...string) error {
	k := d.buildKey(key, more...)
	return d.client.Del(k).Err()
}
