package pool_test

import (
	"testing"

	"github.com/hortbot/hortbot/internal/pkg/pool"
	"gotest.tools/v3/assert"
)

func TestPool(t *testing.T) {
	p := pool.NewPool(func() int { return 42 })
	x := p.Get()
	defer p.Put(x)
	assert.Equal(t, x, 42)
}
