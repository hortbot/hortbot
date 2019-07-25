package rdb

import (
	"time"

	"github.com/go-redis/redis"
)

// KEYS[1] = key
// ARGV[1] = expire time
var checkAndMark = redis.NewScript(`
local exists = redis.pcall('EXISTS', KEYS[1])
if exists == 1 then
	return true
end
redis.pcall('SET', KEYS[1], '1')
redis.pcall('EXPIRE', KEYS[1], ARGV[1])
return false
`)

// KEYS[1] = key
// ARGV[1] = expire time
var checkAndRefresh = redis.NewScript(`
local exists = redis.pcall('EXISTS', KEYS[1])
if exists == 1 then
	redis.pcall('EXPIRE', KEYS[1], ARGV[1])
	return true
end
redis.pcall('SETEX', KEYS[1], '1', ARGV[1])
return false
`)

// KEYS[1] = key
// ARGV[1] = expire time
var markOrDelete = redis.NewScript(`
local exists = redis.pcall('GETSET', KEYS[1], '1')
if exists ~= false then
	redis.pcall('DEL', KEYS[1])
	return true
end
redis.pcall('EXPIRE', KEYS[1], ARGV[1])
return false
`)

// Mark unconditionally sets a value in the database, expiring in the specified number of seconds.
func (d *DB) Mark(seconds int, key string, more ...string) error {
	k := d.buildKey(key, more...)
	dur := time.Duration(seconds) * time.Second
	return d.client.Set(k, "1", dur).Err()
}

// Check checks if a key has been set. It does not modify the value or change its expiration.
func (d *DB) Check(key string, more ...string) (exists bool, err error) {
	k := d.buildKey(key, more...)

	v, err := d.client.Exists(k).Result()
	return v == 1, err
}

// CheckAndMark checks that a key exists, and if it doesn't marks it and sets its expiry.
func (d *DB) CheckAndMark(seconds int, key string, more ...string) (exists bool, err error) {
	k := d.buildKey(key, more...)
	return d.runScript(checkAndMark, k, seconds)
}

// CheckAndRefresh checks that a key exists and refreshes its expiry. If it does not exist, it will be set.
func (d *DB) CheckAndRefresh(seconds int, key string, more ...string) (exists bool, err error) {
	k := d.buildKey(key, more...)
	return d.runScript(checkAndRefresh, k, seconds)
}

// CheckAndDelete checks that a key exists, and removes if it does.
func (d *DB) CheckAndDelete(key string, more ...string) (exists bool, err error) {
	k := d.buildKey(key, more...)
	v, err := d.client.Del(k).Result()
	return v == 1, err
}

// MarkOrDelete marks a key, or deletes it if already present.
func (d *DB) MarkOrDelete(seconds int, key string, more ...string) (exists bool, err error) {
	k := d.buildKey(key, more...)
	return d.runScript(markOrDelete, k, seconds)
}
