// Package docker manages temporary docker containers.
package docker

import (
	"time"

	"github.com/ory/dockertest/v3"
)

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

func (c *Container) Start() (retErr error) {
	pool, err := dockertest.NewPool("")
	if err != nil {
		return err
	}
	c.pool = pool

	if c.ReadyMaxWait != 0 {
		c.pool.MaxWait = c.ReadyMaxWait
	}

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

func (c *Container) Cleanup() {
	if c.resource != nil {
		_ = c.pool.Purge(c.resource)
	}
}

func (c *Container) GetHostPort(portID string) string {
	return c.resource.GetHostPort(portID)
}
