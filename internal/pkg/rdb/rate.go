package rdb

import "github.com/go-redis/redis/v7"

// This script implements a sliding-window rate limiter using a sorted set.
// Set items and their scores are timestamps, and querying the set for the
// current window just means starting somewhere in the set and counting.
//
// 1) Get the current time at the finest grain in redis, microseconds.
// 2) Calculate the beginning of the current window relative to the current time.
// 3) Remove all timestamps in the set which are outside the window.
// 4) If the number of timestamps in the set exceed the limit, then disallow the request.
//    Otherwise, add the current timestamp to the set, and allow the request.
//    Also mark the set to expire after the window that starts at the current time,
//    since if the data isn't read by then, the set won't contain any useful data.
var rateLimit = redis.NewScript(`
local key = KEYS[1]
local limit = tonumber(ARGV[1])
local window_secs = tonumber(ARGV[2])

local redistime = redis.call("TIME")
local now = redistime[1] * 1e6 + redistime[2] -- In microseconds.

local window_start = now - window_secs * 1e6
redis.call("ZREMRANGEBYSCORE", key, "-inf", window_start)

local count = redis.call("ZCARD", key)
if count >= limit then
	return 0
end

redis.call("ZADD", key, "NX", now, now)
redis.call("EXPIRE", key, window_secs + 1)
return 1
`)

// RateLimit rate limits an action using a sliding-window rate limiter, where
// "limit" events can occur within a "windowSecs" seconds long window.
// Rate limiting is accurate to the microsecond, as it does not use EXPIRE for
// something like a token bucket.
//
// This may be pulled out into its own library at some point since it's
// generally helpful.
func (d *DB) RateLimit(limit int, windowSecs int, key string, more ...string) (allowed bool, err error) {
	if limit <= 0 || windowSecs <= 0 {
		return false, nil
	}

	k := d.buildKey(key, more...)
	return d.runScript(rateLimit, k, limit, windowSecs)
}
