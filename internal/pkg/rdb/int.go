package rdb

// GetInt64 returns an integer value with the specified key. If it does not exist, the value will zero.
func (d *DB) GetInt64(key string, more ...string) (int64, error) {
	k := d.buildKey(key, more...)
	return d.client.Get(k).Int64()
}

// Increment increments an integer. If the value d dnot previously exist, it will be incremented to 1.
func (d *DB) Increment(key string, more ...string) (int64, error) {
	k := d.buildKey(key, more...)
	return d.client.Incr(k).Result()
}
