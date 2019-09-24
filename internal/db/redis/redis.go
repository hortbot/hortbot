package redis

import "github.com/go-redis/redis/v7"

type DB struct {
	client *redis.Client
}

func New(client *redis.Client) *DB {
	return &DB{
		client: client,
	}
}

func (db *DB) Close() error {
	return db.client.Close()
}
