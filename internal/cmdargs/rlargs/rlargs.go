// Package rlargs processes rate limiting arguments.
package rlargs

import (
	"context"
	"time"

	"github.com/hortbot/hortbot/internal/db/redis"
)

type RateLimit struct {
	RateLimitSlow   int           `long:"rate-limit-slow" env:"HB_RATE_LIMIT_RATE" description:"Message allowed per rate limit period (slow)"`
	RateLimitFast   int           `long:"rate-limit-fast" env:"HB_RATE_LIMIT_RATE" description:"Message allowed per rate limit period (fast)"`
	RateLimitPeriod time.Duration `long:"rate-limit-period" env:"HB_RATE_LIMIT_PERIOD" description:"Rate limit period"`
}

var DefaultRateLimit = RateLimit{
	RateLimitSlow:   15,
	RateLimitFast:   80,
	RateLimitPeriod: 30 * time.Second,
}

func (args *RateLimit) SendMessageAllowed(ctx context.Context, rdb *redis.DB, origin, target string) (bool, error) {
	return rdb.SendMessageAllowed(ctx, origin, target, args.RateLimitSlow, args.RateLimitFast, args.RateLimitPeriod)
}
