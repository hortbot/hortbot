package redis

import (
	"context"
	"testing"
	"time"

	"github.com/hortbot/hortbot/internal/pkg/testutil/miniredistest"
	"gotest.tools/v3/assert"
)

func TestMarkThenCheck(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()

	s, c, cleanup, err := miniredistest.New()
	assert.NilError(t, err)
	defer cleanup()

	exists, err := check(ctx, c, "#foobar")
	assert.NilError(t, err)
	assert.Assert(t, !exists)

	s.FastForward(time.Second)

	assert.NilError(t, mark(ctx, c, "#foobar", 10*time.Second))

	s.FastForward(time.Second)

	exists, err = check(ctx, c, "#foobar")
	assert.NilError(t, err)
	assert.Assert(t, exists)

	s.FastForward(10 * time.Second)

	exists, err = check(ctx, c, "#foobar")
	assert.NilError(t, err)
	assert.Assert(t, !exists)

	s.FastForward(time.Second)

	exists, err = check(ctx, c, "#foobar")
	assert.NilError(t, err)
	assert.Assert(t, !exists)
}

func TestCheckAndMarkThenCheck(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()

	s, c, cleanup, err := miniredistest.New()
	assert.NilError(t, err)
	defer cleanup()

	exists, err := check(ctx, c, "#foobar")
	assert.NilError(t, err)
	assert.Assert(t, !exists)

	s.FastForward(time.Second)

	exists, err = checkAndMark(ctx, c, "#foobar", 10*time.Second)
	assert.NilError(t, err)
	assert.Assert(t, !exists)

	s.FastForward(time.Second)

	exists, err = check(ctx, c, "#foobar")
	assert.NilError(t, err)
	assert.Assert(t, exists)

	s.FastForward(10 * time.Second)

	exists, err = check(ctx, c, "#foobar")
	assert.NilError(t, err)
	assert.Assert(t, !exists)

	s.FastForward(time.Second)

	exists, err = check(ctx, c, "#foobar")
	assert.NilError(t, err)
	assert.Assert(t, !exists)
}

func TestCheckAndMarkTwiceThenCheck(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()

	s, c, cleanup, err := miniredistest.New()
	assert.NilError(t, err)
	defer cleanup()

	exists, err := check(ctx, c, "#foobar")
	assert.NilError(t, err)
	assert.Assert(t, !exists)

	s.FastForward(time.Second)

	exists, err = checkAndMark(ctx, c, "#foobar", 10*time.Second)
	assert.NilError(t, err)
	assert.Assert(t, !exists)

	s.FastForward(5 * time.Second)

	exists, err = checkAndMark(ctx, c, "#foobar", 10*time.Second)
	assert.NilError(t, err)
	assert.Assert(t, exists)

	s.FastForward(10 * time.Second)

	exists, err = check(ctx, c, "#foobar")
	assert.NilError(t, err)
	assert.Assert(t, !exists)

	s.FastForward(time.Second)

	exists, err = check(ctx, c, "#foobar")
	assert.NilError(t, err)
	assert.Assert(t, !exists)
}

func TestMarkAndDelete(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()

	s, c, cleanup, err := miniredistest.New()
	assert.NilError(t, err)
	defer cleanup()

	exists, err := check(ctx, c, "#foobar")
	assert.NilError(t, err)
	assert.Assert(t, !exists)

	s.FastForward(time.Second)

	assert.NilError(t, mark(ctx, c, "#foobar", 10*time.Second))

	s.FastForward(time.Second)

	exists, err = check(ctx, c, "#foobar")
	assert.NilError(t, err)
	assert.Assert(t, exists)

	s.FastForward(time.Second)

	exists, err = checkAndDelete(ctx, c, "#foobar")
	assert.NilError(t, err)
	assert.Assert(t, exists)

	s.FastForward(time.Second)

	exists, err = check(ctx, c, "#foobar")
	assert.NilError(t, err)
	assert.Assert(t, !exists)
}

func TestRefresh(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()

	s, c, cleanup, err := miniredistest.New()
	assert.NilError(t, err)
	defer cleanup()

	exists, err := check(ctx, c, "#foobar")
	assert.NilError(t, err)
	assert.Assert(t, !exists)

	s.FastForward(time.Second)

	exists, err = checkAndMark(ctx, c, "#foobar", 10*time.Second)
	assert.NilError(t, err)
	assert.Assert(t, !exists)

	s.FastForward(5 * time.Second)

	exists, err = checkAndRefresh(ctx, c, "#foobar", 15*time.Second)
	assert.NilError(t, err)
	assert.Assert(t, exists)

	s.FastForward(10 * time.Second)

	exists, err = check(ctx, c, "#foobar")
	assert.NilError(t, err)
	assert.Assert(t, exists)

	s.FastForward(10 * time.Second)

	exists, err = check(ctx, c, "#foobar")
	assert.NilError(t, err)
	assert.Assert(t, !exists)
}

func TestMarkOrDelete(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()

	s, c, cleanup, err := miniredistest.New()
	assert.NilError(t, err)
	defer cleanup()

	exists, err := check(ctx, c, "#foobar")
	assert.NilError(t, err)
	assert.Assert(t, !exists)

	s.FastForward(time.Second)

	exists, err = markOrDelete(ctx, c, "#foobar", 10*time.Second)
	assert.NilError(t, err)
	assert.Assert(t, !exists)

	s.FastForward(time.Second)

	exists, err = check(ctx, c, "#foobar")
	assert.NilError(t, err)
	assert.Assert(t, exists)

	s.FastForward(time.Second)

	exists, err = markOrDelete(ctx, c, "#foobar", 10*time.Second)
	assert.NilError(t, err)
	assert.Assert(t, exists)

	s.FastForward(time.Second)

	exists, err = check(ctx, c, "#foobar")
	assert.NilError(t, err)
	assert.Assert(t, !exists)
}
