package redistest

import (
	"github.com/go-redis/redis/v7"
	"github.com/ory/dockertest"
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
			pool.Purge(resource) //nolint:errcheck
		}
	}()

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
		pool.Purge(resource) //nolint:errcheck
	}, nil
}
