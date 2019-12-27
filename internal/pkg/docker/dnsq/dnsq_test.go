package dnsq_test

import (
	"testing"

	"github.com/hortbot/hortbot/internal/pkg/docker/dnsq"
	"gotest.tools/v3/assert"
)

func TestNew(t *testing.T) {
	addr, cleanup, err := dnsq.New()
	assert.NilError(t, err)
	assert.Assert(t, cleanup != nil)
	cleanup()
	assert.Assert(t, addr != "", "got address: %s", addr)
}
