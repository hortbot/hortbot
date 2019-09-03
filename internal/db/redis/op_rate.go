package redis

import (
	"time"

	"github.com/go-redis/redis/v7"
)

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
var scriptRateLimit = redis.NewScript(`
local key = KEYS[1]
local limit = tonumber(ARGV[1])
local window_microsecs = tonumber(ARGV[2])

local redistime = redis.call("TIME")
local now = (redistime[1] * 1e6) + redistime[2]

local window_start = now - window_microsecs
redis.call("ZREMRANGEBYSCORE", key, "-inf", window_start)

local count = redis.call("ZCARD", key)
if count >= limit then
	return 0
end

redis.call("ZADD", key, "NX", now, now)
redis.call("EXPIRE", key, (window_microsecs / 1e6) + 1)
return 1
`)

// rateLimit rate limits an action using a sliding-window rate limiter, where
// "limit" events can occur within a "window" long window.
// Rate limiting is accurate to the microsecond, as it does not use EXPIRE for
// something like a token bucket.
//
// This may be pulled out into its own library at some point since it's
// generally helpful.
func rateLimit(client redis.Cmdable, key string, limit int, window time.Duration) (allowed bool, err error) {
	windowMicro := int64(window / time.Microsecond)
	if limit <= 0 || windowMicro <= 0 {
		return false, nil
	}

	return scriptRateLimit.Run(client, []string{key}, limit, windowMicro).Bool()
}
