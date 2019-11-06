package redistest

import (
	"github.com/go-redis/redis/v7"
	"github.com/ory/dockertest/v3"
)

func New() (client *redis.Client, cleanup func(), retErr error) {
	pool, err := dockertest.NewPool("")
	if err != nil {
		return nil, nil, err
	}

	resource, err := pool.Run("redis", "latest", nil)
	if err != nil {
		return nil, nil, err
	}

	defer func() {
		if retErr != nil {
			_ = pool.Purge(resource)
		}
	}()

	// Ensure the container is cleaned up, even if the process exits.
	if err := resource.Expire(300); err != nil {
		return nil, nil, err
	}

	err = pool.Retry(func() error {
		client = redis.NewClient(&redis.Options{
			Addr: resource.GetHostPort("6379/tcp"),
		})

		_, err := client.Ping().Result()
		return err
	})
	if err != nil {
		return nil, nil, err
	}

	defer func() {
		if retErr != nil {
			client.Close()
		}
	}()

	return client, func() {
		client.Close()
		_ = pool.Purge(resource)
	}, nil
}
