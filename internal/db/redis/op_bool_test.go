package redis

import (
	"testing"
	"time"

	"github.com/hortbot/hortbot/internal/pkg/testutil/miniredistest"
	"gotest.tools/v3/assert"
)

func TestMarkThenCheck(t *testing.T) {
	t.Parallel()

	s, c, cleanup, err := miniredistest.New()
	assert.NilError(t, err)
	defer cleanup()

	exists, err := check(c, "#foobar")
	assert.NilError(t, err)
	assert.Assert(t, !exists)

	s.FastForward(time.Second)

	assert.NilError(t, mark(c, "#foobar", 10*time.Second))

	s.FastForward(time.Second)

	exists, err = check(c, "#foobar")
	assert.NilError(t, err)
	assert.Assert(t, exists)

	s.FastForward(10 * time.Second)

	exists, err = check(c, "#foobar")
	assert.NilError(t, err)
	assert.Assert(t, !exists)

	s.FastForward(time.Second)

	exists, err = check(c, "#foobar")
	assert.NilError(t, err)
	assert.Assert(t, !exists)
}

func TestCheckAndMarkThenCheck(t *testing.T) {
	t.Parallel()

	s, c, cleanup, err := miniredistest.New()
	assert.NilError(t, err)
	defer cleanup()

	exists, err := check(c, "#foobar")
	assert.NilError(t, err)
	assert.Assert(t, !exists)

	s.FastForward(time.Second)

	exists, err = checkAndMark(c, "#foobar", 10*time.Second)
	assert.NilError(t, err)
	assert.Assert(t, !exists)

	s.FastForward(time.Second)

	exists, err = check(c, "#foobar")
	assert.NilError(t, err)
	assert.Assert(t, exists)

	s.FastForward(10 * time.Second)

	exists, err = check(c, "#foobar")
	assert.NilError(t, err)
	assert.Assert(t, !exists)

	s.FastForward(time.Second)

	exists, err = check(c, "#foobar")
	assert.NilError(t, err)
	assert.Assert(t, !exists)
}

func TestCheckAndMarkTwiceThenCheck(t *testing.T) {
	t.Parallel()

	s, c, cleanup, err := miniredistest.New()
	assert.NilError(t, err)
	defer cleanup()

	exists, err := check(c, "#foobar")
	assert.NilError(t, err)
	assert.Assert(t, !exists)

	s.FastForward(time.Second)

	exists, err = checkAndMark(c, "#foobar", 10*time.Second)
	assert.NilError(t, err)
	assert.Assert(t, !exists)

	s.FastForward(5 * time.Second)

	exists, err = checkAndMark(c, "#foobar", 10*time.Second)
	assert.NilError(t, err)
	assert.Assert(t, exists)

	s.FastForward(10 * time.Second)

	exists, err = check(c, "#foobar")
	assert.NilError(t, err)
	assert.Assert(t, !exists)

	s.FastForward(time.Second)

	exists, err = check(c, "#foobar")
	assert.NilError(t, err)
	assert.Assert(t, !exists)
}

func TestMarkAndDelete(t *testing.T) {
	t.Parallel()

	s, c, cleanup, err := miniredistest.New()
	assert.NilError(t, err)
	defer cleanup()

	exists, err := check(c, "#foobar")
	assert.NilError(t, err)
	assert.Assert(t, !exists)

	s.FastForward(time.Second)

	assert.NilError(t, mark(c, "#foobar", 10*time.Second))

	s.FastForward(time.Second)

	exists, err = check(c, "#foobar")
	assert.NilError(t, err)
	assert.Assert(t, exists)

	s.FastForward(time.Second)

	exists, err = checkAndDelete(c, "#foobar")
	assert.NilError(t, err)
	assert.Assert(t, exists)

	s.FastForward(time.Second)

	exists, err = check(c, "#foobar")
	assert.NilError(t, err)
	assert.Assert(t, !exists)
}

func TestRefresh(t *testing.T) {
	t.Parallel()

	s, c, cleanup, err := miniredistest.New()
	assert.NilError(t, err)
	defer cleanup()

	exists, err := check(c, "#foobar")
	assert.NilError(t, err)
	assert.Assert(t, !exists)

	s.FastForward(time.Second)

	exists, err = checkAndMark(c, "#foobar", 10*time.Second)
	assert.NilError(t, err)
	assert.Assert(t, !exists)

	s.FastForward(5 * time.Second)

	exists, err = checkAndRefresh(c, "#foobar", 15*time.Second)
	assert.NilError(t, err)
	assert.Assert(t, exists)

	s.FastForward(10 * time.Second)

	exists, err = check(c, "#foobar")
	assert.NilError(t, err)
	assert.Assert(t, exists)

	s.FastForward(10 * time.Second)

	exists, err = check(c, "#foobar")
	assert.NilError(t, err)
	assert.Assert(t, !exists)
}

func TestMarkOrDelete(t *testing.T) {
	t.Parallel()

	s, c, cleanup, err := miniredistest.New()
	assert.NilError(t, err)
	defer cleanup()

	exists, err := check(c, "#foobar")
	assert.NilError(t, err)
	assert.Assert(t, !exists)

	s.FastForward(time.Second)

	exists, err = markOrDelete(c, "#foobar", 10*time.Second)
	assert.NilError(t, err)
	assert.Assert(t, !exists)

	s.FastForward(time.Second)

	exists, err = check(c, "#foobar")
	assert.NilError(t, err)
	assert.Assert(t, exists)

	s.FastForward(time.Second)

	exists, err = markOrDelete(c, "#foobar", 10*time.Second)
	assert.NilError(t, err)
	assert.Assert(t, exists)

	s.FastForward(time.Second)

	exists, err = check(c, "#foobar")
	assert.NilError(t, err)
	assert.Assert(t, !exists)
}
