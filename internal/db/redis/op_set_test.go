package redis

import (
	"testing"

	"github.com/hortbot/hortbot/internal/pkg/testutil/miniredistest"
	"gotest.tools/v3/assert"
)

func TestSetEmpty(t *testing.T) {
	t.Parallel()

	_, c, cleanup, err := miniredistest.New()
	assert.NilError(t, err)
	defer cleanup()

	l, err := setLen(c, "foo")
	assert.NilError(t, err)
	assert.Equal(t, l, int64(0))

	s, ok, err := setPop(c, "foo")
	assert.NilError(t, err)
	assert.Equal(t, s, "")
	assert.Assert(t, !ok)

	assert.NilError(t, setClear(c, "foo"))

	l, err = setLen(c, "foo")
	assert.NilError(t, err)
	assert.Equal(t, l, int64(0))
}

func TestSetAdd(t *testing.T) {
	t.Parallel()

	_, c, cleanup, err := miniredistest.New()
	assert.NilError(t, err)
	defer cleanup()

	l, err := setLen(c, "foobar")
	assert.NilError(t, err)
	assert.Equal(t, l, int64(0))

	assert.NilError(t, setAdd(c, "foobar", "v1"))

	l, err = setLen(c, "foobar")
	assert.NilError(t, err)
	assert.Equal(t, l, int64(1))

	assert.NilError(t, setAdd(c, "foobar", "v2"))

	l, err = setLen(c, "foobar")
	assert.NilError(t, err)
	assert.Equal(t, l, int64(2))

	assert.NilError(t, setAdd(c, "foobar", "v2"))

	l, err = setLen(c, "foobar")
	assert.NilError(t, err)
	assert.Equal(t, l, int64(2))
}

func TestSetClear(t *testing.T) {
	t.Parallel()

	_, c, cleanup, err := miniredistest.New()
	assert.NilError(t, err)
	defer cleanup()

	l, err := setLen(c, "foobar")
	assert.NilError(t, err)
	assert.Equal(t, l, int64(0))

	assert.NilError(t, setAdd(c, "foobar", "v1"))
	assert.NilError(t, setAdd(c, "foobar", "v2"))
	assert.NilError(t, setAdd(c, "foobar", "v3"))

	l, err = setLen(c, "foobar")
	assert.NilError(t, err)
	assert.Equal(t, l, int64(3))

	assert.NilError(t, setClear(c, "foobar"))

	l, err = setLen(c, "foobar")
	assert.NilError(t, err)
	assert.Equal(t, l, int64(0))
}

func TestSetPop(t *testing.T) {
	t.Parallel()

	s, c, cleanup, err := miniredistest.New()
	assert.NilError(t, err)
	defer cleanup()

	s.Seed(311)

	l, err := setLen(c, "foobar")
	assert.NilError(t, err)
	assert.Equal(t, l, int64(0))

	assert.NilError(t, setAdd(c, "foobar", "v1"))
	assert.NilError(t, setAdd(c, "foobar", "v2"))
	assert.NilError(t, setAdd(c, "foobar", "v3"))

	v, ok, err := setPop(c, "foobar")
	assert.NilError(t, err)
	assert.Assert(t, ok)
	assert.Equal(t, v, "v2")

	v, ok, err = setPop(c, "foobar")
	assert.NilError(t, err)
	assert.Assert(t, ok)
	assert.Equal(t, v, "v3")

	v, ok, err = setPop(c, "foobar")
	assert.NilError(t, err)
	assert.Assert(t, ok)
	assert.Equal(t, v, "v1")

	v, ok, err = setPop(c, "foobar")
	assert.NilError(t, err)
	assert.Assert(t, !ok)
	assert.Equal(t, v, "")
}
