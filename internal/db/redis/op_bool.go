package redis

import (
	"time"

	"github.com/go-redis/redis/v7"
)

// KEYS[1] = key
// ARGV[1] = expire time
var scriptCheckAndMark = redis.NewScript(`
local exists = redis.pcall('EXISTS', KEYS[1])
if exists == 1 then
	return 1
end
redis.pcall('SET', KEYS[1], '1')
redis.pcall('EXPIRE', KEYS[1], ARGV[1])
return 0
`)

// KEYS[1] = key
// ARGV[1] = expire time
var scriptCheckAndRefresh = redis.NewScript(`
local exists = redis.pcall('EXISTS', KEYS[1])
if exists == 1 then
	redis.pcall('EXPIRE', KEYS[1], ARGV[1])
	return 1
end
redis.pcall('SETEX', KEYS[1], '1', ARGV[1])
return 0
`)

// KEYS[1] = key
// ARGV[1] = expire time
var scriptMarkOrDelete = redis.NewScript(`
local exists = redis.pcall('GETSET', KEYS[1], '1')
if exists ~= false then
	redis.pcall('DEL', KEYS[1])
	return 1
end
redis.pcall('EXPIRE', KEYS[1], ARGV[1])
return 0
`)

// mark unconditionally sets a value in the database, expiring in the specified number of seconds.
func mark(client redis.Cmdable, key string, expiry time.Duration) error {
	return client.Set(key, "1", expiry).Err()
}

// check checks if a key has been set. It does not modify the value or change its expiration.
func check(client redis.Cmdable, key string) (exists bool, err error) {
	v, err := client.Exists(key).Result()
	return v == 1, err
}

// checkAndMark checks that a key exists, and if it doesn't marks it and sets its expiry.
func checkAndMark(client redis.Cmdable, key string, expiry time.Duration) (exists bool, err error) {
	secs := int64(expiry / time.Second)
	return scriptCheckAndMark.Run(client, []string{key}, secs).Bool()
}

// checkAndRefresh checks that a key exists and refreshes its expiry. If it does not exist, it will be set.
func checkAndRefresh(client redis.Cmdable, key string, expiry time.Duration) (exists bool, err error) {
	secs := int64(expiry / time.Second)
	return scriptCheckAndRefresh.Run(client, []string{key}, secs).Bool()
}

// checkAndDelete checks that a key exists, and removes if it does.
func checkAndDelete(client redis.Cmdable, key string) (exists bool, err error) {
	v, err := client.Del(key).Result()
	return v == 1, err
}

// markOrDelete marks a key, or deletes it if already present.
func markOrDelete(client redis.Cmdable, key string, expiry time.Duration) (exists bool, err error) {
	secs := int64(expiry / time.Second)
	return scriptMarkOrDelete.Run(client, []string{key}, secs).Bool()
}
