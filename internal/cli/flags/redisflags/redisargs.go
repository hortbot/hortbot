// Package redisflags processes redis-related flags.
package redisflags

import (
	goredis "github.com/go-redis/redis/v8"
	"github.com/hortbot/hortbot/internal/db/redis"
)

// Redis contains redis flags.
type Redis struct {
	Addr string `long:"redis-addr" env:"HB_REDIS_ADDR" description:"Redis address" required:"true"`
}

// Default contains the default flags. Make a copy of this, do not reuse.
var Default = Redis{}

// Client creates a new redis client from the configured flags.
func (args *Redis) Client() *redis.DB {
	return redis.New(goredis.NewClient(&goredis.Options{
		Addr: args.Addr,
	}))
}
