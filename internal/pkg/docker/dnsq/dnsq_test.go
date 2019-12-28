package dnsq_test

import (
	"testing"

	"github.com/hortbot/hortbot/internal/pkg/docker/dnsq"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/env"
)

func TestNew(t *testing.T) {
	addr, cleanup, err := dnsq.New()
	assert.NilError(t, err)
	assert.Assert(t, cleanup != nil)
	cleanup()
	assert.Assert(t, addr != "", "got address: %s", addr)
}

func TestNewBadDocker(t *testing.T) {
	defer env.Patch(t, "DOCKER_URL", "tcp://[[[[[")()

	_, _, err := dnsq.New()
	assert.ErrorContains(t, err, "invalid endpoint")
}
