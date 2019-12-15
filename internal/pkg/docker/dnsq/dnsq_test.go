package dnsq_test

import (
	"testing"

	"github.com/hortbot/hortbot/internal/pkg/docker/dnsq"
	"gotest.tools/v3/assert"
)

func TestNew(t *testing.T) {
	if testing.Short() {
		t.Skip("requires starting a docker container")
	}

	addr, cleanup, err := dnsq.New()
	assert.NilError(t, err)
	assert.Assert(t, cleanup != nil)
	cleanup()
	assert.Assert(t, addr != "", "got address: %s", addr)
}
