// Package redisargs processes redis arguments.
package redisargs

import (
	goredis "github.com/go-redis/redis/v7"
	"github.com/hortbot/hortbot/internal/db/redis"
)

type Redis struct {
	RedisAddr string `long:"redis-addr" env:"HB_REDIS_ADDR" description:"Redis address" required:"true"`
}

var DefaultRedis = Redis{}

func (args *Redis) RedisClient() *redis.DB {
	return redis.New(goredis.NewClient(&goredis.Options{
		Addr: args.RedisAddr,
	}))
}
