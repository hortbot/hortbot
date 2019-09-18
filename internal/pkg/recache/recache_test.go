package recache_test

import (
	"regexp"
	"testing"

	"github.com/hortbot/hortbot/internal/pkg/recache"
	"gotest.tools/v3/assert"
)

const pattern = `.*\Qwinlan\E.*`

func TestCompile(t *testing.T) {
	c := recache.New()

	r, err := c.Compile(pattern)
	assert.NilError(t, err)
	assert.Assert(t, r != nil)
	assert.Equal(t, r.String(), `(?i)`+pattern)

	r2, err := c.Compile(pattern)
	assert.NilError(t, err)
	assert.Assert(t, r2 == r)
}

func BenchmarkCompile(b *testing.B) {
	c := recache.New()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = c.Compile(pattern)
	}
}

func BenchmarkCompileNative(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, _ = regexp.Compile(pattern)
	}
}
