// Package docker manages temporary docker containers.
package docker

import (
	"time"

	"github.com/ory/dockertest/v3"
)

// Container defines a docker container.
type Container struct {
	Repository string
	Tag        string
	Cmd        []string
	Env        []string
	Mounts     []string

	Ready        func(*Container) error
	ReadyMaxWait time.Duration

	ExpirySecs uint

	pool     *dockertest.Pool
	resource *dockertest.Resource
}

// Start starts the container.
func (c *Container) Start() (retErr error) {
	pool, err := dockertest.NewPool("")
	if err != nil {
		return err
	}
	c.pool = pool
	c.pool.MaxWait = c.ReadyMaxWait

	resource, err := pool.RunWithOptions(&dockertest.RunOptions{
		Repository: c.Repository,
		Tag:        c.Tag,
		Cmd:        c.Cmd,
		Env:        c.Env,
		Mounts:     c.Mounts,
	})
	if err != nil {
		return err
	}
	c.resource = resource

	defer func() {
		if retErr != nil {
			c.resource = nil
			_ = pool.Purge(resource)
		}
	}()

	if c.ExpirySecs != 0 {
		if err := resource.Expire(c.ExpirySecs); err != nil {
			return err
		}
	}

	return pool.Retry(func() error {
		return c.Ready(c)
	})
}

// Cleanup cleans up the docker container, stopping and removing it.
func (c *Container) Cleanup() {
	if c.resource != nil {
		_ = c.pool.Purge(c.resource)
	}
}

// GetHostPort gets the correct host and port pair for the specified port after
// being forwarded. For example, GetHostPort("5432/tcp") could return
// "localhost:13263".
func (c *Container) GetHostPort(portID string) string {
	return c.resource.GetHostPort(portID)
}
