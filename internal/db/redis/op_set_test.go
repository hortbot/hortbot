package redis

import (
	"context"
	"testing"
	"time"

	"github.com/hortbot/hortbot/internal/pkg/testutil/miniredistest"
	"gotest.tools/v3/assert"
)

func TestSetEmpty(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, c, cleanup, err := miniredistest.New()
	assert.NilError(t, err)
	defer cleanup()

	l, err := setLen(ctx, c, "foo")
	assert.NilError(t, err)
	assert.Equal(t, l, int64(0))

	s, ok, err := setPop(ctx, c, "foo")
	assert.NilError(t, err)
	assert.Equal(t, s, "")
	assert.Assert(t, !ok)

	assert.NilError(t, setClear(ctx, c, "foo"))

	l, err = setLen(ctx, c, "foo")
	assert.NilError(t, err)
	assert.Equal(t, l, int64(0))
}

func TestSetAdd(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, c, cleanup, err := miniredistest.New()
	assert.NilError(t, err)
	defer cleanup()

	l, err := setLen(ctx, c, "foobar")
	assert.NilError(t, err)
	assert.Equal(t, l, int64(0))

	assert.NilError(t, setAdd(ctx, c, "foobar", "v1"))

	l, err = setLen(ctx, c, "foobar")
	assert.NilError(t, err)
	assert.Equal(t, l, int64(1))

	assert.NilError(t, setAdd(ctx, c, "foobar", "v2"))

	l, err = setLen(ctx, c, "foobar")
	assert.NilError(t, err)
	assert.Equal(t, l, int64(2))

	assert.NilError(t, setAdd(ctx, c, "foobar", "v2"))

	l, err = setLen(ctx, c, "foobar")
	assert.NilError(t, err)
	assert.Equal(t, l, int64(2))
}

func TestSetClear(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, c, cleanup, err := miniredistest.New()
	assert.NilError(t, err)
	defer cleanup()

	l, err := setLen(ctx, c, "foobar")
	assert.NilError(t, err)
	assert.Equal(t, l, int64(0))

	assert.NilError(t, setAdd(ctx, c, "foobar", "v1"))
	assert.NilError(t, setAdd(ctx, c, "foobar", "v2"))
	assert.NilError(t, setAdd(ctx, c, "foobar", "v3"))

	l, err = setLen(ctx, c, "foobar")
	assert.NilError(t, err)
	assert.Equal(t, l, int64(3))

	assert.NilError(t, setClear(ctx, c, "foobar"))

	l, err = setLen(ctx, c, "foobar")
	assert.NilError(t, err)
	assert.Equal(t, l, int64(0))
}

func TestSetPop(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	s, c, cleanup, err := miniredistest.New()
	assert.NilError(t, err)
	defer cleanup()

	s.Seed(311)

	l, err := setLen(ctx, c, "foobar")
	assert.NilError(t, err)
	assert.Equal(t, l, int64(0))

	assert.NilError(t, setAdd(ctx, c, "foobar", "v1"))
	assert.NilError(t, setAdd(ctx, c, "foobar", "v2"))
	assert.NilError(t, setAdd(ctx, c, "foobar", "v3"))

	v, ok, err := setPop(ctx, c, "foobar")
	assert.NilError(t, err)
	assert.Assert(t, ok)
	assert.Equal(t, v, "v2")

	v, ok, err = setPop(ctx, c, "foobar")
	assert.NilError(t, err)
	assert.Assert(t, ok)
	assert.Equal(t, v, "v3")

	v, ok, err = setPop(ctx, c, "foobar")
	assert.NilError(t, err)
	assert.Assert(t, ok)
	assert.Equal(t, v, "v1")

	v, ok, err = setPop(ctx, c, "foobar")
	assert.NilError(t, err)
	assert.Assert(t, !ok)
	assert.Equal(t, v, "")
}

func TestSetPopN(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	s, c, cleanup, err := miniredistest.New()
	assert.NilError(t, err)
	defer cleanup()

	s.Seed(311)

	l, err := setLen(ctx, c, "foobar")
	assert.NilError(t, err)
	assert.Equal(t, l, int64(0))

	assert.NilError(t, setAdd(ctx, c, "foobar", "v1"))
	assert.NilError(t, setAdd(ctx, c, "foobar", "v2"))
	assert.NilError(t, setAdd(ctx, c, "foobar", "v3"))
	assert.NilError(t, setAdd(ctx, c, "foobar", "v4"))

	v, err := setPopN(ctx, c, "foobar", 3)
	assert.NilError(t, err)
	assert.DeepEqual(t, v, []string{"v4", "v2", "v3"})

	v, err = setPopN(ctx, c, "foobar", 2)
	assert.NilError(t, err)
	assert.DeepEqual(t, v, []string{"v1"})

	v, err = setPopN(ctx, c, "foobar", 2)
	assert.NilError(t, err)
	assert.DeepEqual(t, v, []string{})
}
