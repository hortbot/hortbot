// Package redis implements a type-safe redis database client.
package redis

import "github.com/redis/go-redis/v9"

// DB is a redis client wrapper, consolidating the common schema in one place.
type DB struct {
	client *redis.Client
}

// New creates a new redis client DB wrapper. The user is expected to close
// the client itself; DB is just a wrapper and does not manage the connection.
func New(client *redis.Client) *DB {
	return &DB{
		client: client,
	}
}

func ignoreRedisNil(err error) error {
	if err == redis.Nil {
		return nil
	}
	return err
}
