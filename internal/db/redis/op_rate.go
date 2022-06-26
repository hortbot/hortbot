package redis

import (
	"context"
	"time"

	"github.com/go-redis/redis/v8"
)

// This script implements a sliding-window rate limiter using a sorted set.
// Set items and their scores are timestamps, and querying the set for the
// current window just means starting somewhere in the set and counting.
//
//  1. Get the current time at the finest grain in redis, microseconds.
//  2. Calculate the beginning of the current window relative to the current time.
//  3. Remove all timestamps in the set which are outside the window.
//  4. If the number of timestamps in the set exceed the limit, then disallow the request.
//     Otherwise, add the current timestamp to the set, and allow the request.
//     Also mark the set to expire after the window that starts at the current time,
//     since if the data isn't read by then, the set won't contain any useful data.
//
// KEYS[1] = overall key
// KEYS[2] = a key which is true when only a fast token is needed
// ARGV[1] = window in microseconds
// ARGV[2] = slow limit
// ARGV[3] = fast limit
var scriptRateLimit = redis.NewScript(`
local slow_key = KEYS[1] .. ":slow"
local fast_key = KEYS[1] .. ":fast"
local window_microsecs = tonumber(ARGV[1])
local slow_limit = tonumber(ARGV[2])
local fast_limit = tonumber(ARGV[3])
local only_fast = redis.call("GET", KEYS[2]) == "1"

local redistime = redis.call("TIME")
local now = (redistime[1] * 1e6) + redistime[2]
local window_start = now - window_microsecs

local function block(k, limit)
	redis.call("ZREMRANGEBYSCORE", k, "-inf", window_start)
	return redis.call("ZCARD", k) >= limit
end

local function add(k)
	redis.call("ZADD", k, "NX", now, now)
	redis.call("EXPIRE", k, (window_microsecs / 1e6) + 1)
end

if block(fast_key, fast_limit) then
	return 0
end

if only_fast then
	add(fast_key)
	return 1
end

if block(slow_key, slow_limit) then
	return 0
end

add(fast_key)
add(slow_key)

return 1
`)

func rateLimit(ctx context.Context, client redis.Cmdable, key string, window time.Duration, slowLimit, fastLimit int, onlyFastKey string) (allowed bool, err error) {
	windowMicro := int64(window / time.Microsecond)
	if windowMicro <= 0 {
		return false, nil
	}
	return scriptRateLimit.Run(ctx, client, []string{key, onlyFastKey}, windowMicro, slowLimit, fastLimit).Bool()
}
