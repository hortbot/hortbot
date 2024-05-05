// Package docker manages temporary docker containers.
package docker

import (
	"slices"
	"time"

	"github.com/ory/dockertest/v3"
	"github.com/ory/dockertest/v3/docker"
)

// Container defines a docker container.
type Container struct {
	Repository string
	Tag        string
	Cmd        []string
	Env        []string
	Mounts     []string
	Ports      []string

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
	}, func(hc *docker.HostConfig) {
		// There's some sort of bug in dockertest, the docker API, or the docker daemon
		// where auto-publishing ports leads to some containers sharing the same ports
		// on the host, mixing up connections. For example:
		//
		//     0.0.0.0:33179->5432/tcp, :::33177->5432/tcp   unruffled_agnesi
		//     0.0.0.0:33181->5432/tcp, :::33179->5432/tcp   determined_bartik
		//     0.0.0.0:33184->5432/tcp, :::33182->5432/tcp   trusting_heyrovsky
		//     0.0.0.0:33183->5432/tcp, :::33181->5432/tcp   ecstatic_lehmann
		//     0.0.0.0:33178->5432/tcp, :::33176->5432/tcp   charming_ptolemy
		//     0.0.0.0:33180->5432/tcp, :::33178->5432/tcp   xenodochial_sanderson
		//
		// It seems to not happen when we manually map things to explicitly 0.0.0.0.
		hc.PublishAllPorts = false
		hc.PortBindings = make(map[docker.Port][]docker.PortBinding)
		for _, port := range c.Ports {
			p := docker.Port(port)
			hc.PortBindings[p] = []docker.PortBinding{{HostIP: "0.0.0.0", HostPort: "0/" + p.Proto()}}
		}
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
	if !slices.Contains(c.Ports, portID) {
		panic(portID + " not in Ports")
	}
	return c.resource.GetHostPort(portID)
}
