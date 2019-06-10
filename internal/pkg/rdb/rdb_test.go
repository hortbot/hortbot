package rdb_test

import (
	"testing"
	"time"

	"github.com/hortbot/hortbot/internal/pkg/rdb"
	"github.com/hortbot/hortbot/internal/pkg/testutil/miniredistest"
	"gotest.tools/assert"
)

func TestMarkThenCheck(t *testing.T) {
	t.Parallel()

	s, c, cleanup, err := miniredistest.New()
	assert.NilError(t, err)
	defer cleanup()

	db, err := rdb.New(c)
	assert.NilError(t, err)

	seen, err := db.Check("#foobar", "something")
	assert.NilError(t, err)
	assert.Assert(t, !seen)

	s.FastForward(time.Second)

	assert.NilError(t, db.Mark(10, "#foobar", "something"))

	s.FastForward(time.Second)

	seen, err = db.Check("#foobar", "something")
	assert.NilError(t, err)
	assert.Assert(t, seen)

	s.FastForward(10 * time.Second)

	seen, err = db.Check("#foobar", "something")
	assert.NilError(t, err)
	assert.Assert(t, !seen)

	s.FastForward(time.Second)

	seen, err = db.Check("#foobar", "something")
	assert.NilError(t, err)
	assert.Assert(t, !seen)
}

func TestCheckAndMarkThenCheck(t *testing.T) {
	t.Parallel()

	s, c, cleanup, err := miniredistest.New()
	assert.NilError(t, err)
	defer cleanup()

	db, err := rdb.New(c)
	assert.NilError(t, err)

	seen, err := db.Check("#foobar", "something")
	assert.NilError(t, err)
	assert.Assert(t, !seen)

	s.FastForward(time.Second)

	seen, err = db.CheckAndMark(10, "#foobar", "something")
	assert.NilError(t, err)
	assert.Assert(t, !seen)

	s.FastForward(time.Second)

	seen, err = db.Check("#foobar", "something")
	assert.NilError(t, err)
	assert.Assert(t, seen)

	s.FastForward(10 * time.Second)

	seen, err = db.Check("#foobar", "something")
	assert.NilError(t, err)
	assert.Assert(t, !seen)

	s.FastForward(time.Second)

	seen, err = db.Check("#foobar", "something")
	assert.NilError(t, err)
	assert.Assert(t, !seen)
}

func TestMarkAndDelete(t *testing.T) {
	t.Parallel()

	s, c, cleanup, err := miniredistest.New()
	assert.NilError(t, err)
	defer cleanup()

	db, err := rdb.New(c)
	assert.NilError(t, err)

	seen, err := db.Check("#foobar", "something")
	assert.NilError(t, err)
	assert.Assert(t, !seen)

	s.FastForward(time.Second)

	assert.NilError(t, db.Mark(10, "#foobar", "something"))

	s.FastForward(time.Second)

	seen, err = db.Check("#foobar", "something")
	assert.NilError(t, err)
	assert.Assert(t, seen)

	s.FastForward(time.Second)

	seen, err = db.CheckAndDelete("#foobar", "something")
	assert.NilError(t, err)
	assert.Assert(t, seen)

	s.FastForward(time.Second)

	seen, err = db.Check("#foobar", "something")
	assert.NilError(t, err)
	assert.Assert(t, !seen)
}

func TestRefresh(t *testing.T) {
	t.Parallel()

	s, c, cleanup, err := miniredistest.New()
	assert.NilError(t, err)
	defer cleanup()

	db, err := rdb.New(c)
	assert.NilError(t, err)

	seen, err := db.Check("#foobar", "something")
	assert.NilError(t, err)
	assert.Assert(t, !seen)

	s.FastForward(time.Second)

	seen, err = db.CheckAndMark(10, "#foobar", "something")
	assert.NilError(t, err)
	assert.Assert(t, !seen)

	s.FastForward(5 * time.Second)

	seen, err = db.CheckAndRefresh(15, "#foobar", "something")
	assert.NilError(t, err)
	assert.Assert(t, seen)

	s.FastForward(10 * time.Second)

	seen, err = db.Check("#foobar", "something")
	assert.NilError(t, err)
	assert.Assert(t, seen)

	s.FastForward(10 * time.Second)

	seen, err = db.Check("#foobar", "something")
	assert.NilError(t, err)
	assert.Assert(t, !seen)
}

func TestPrefix(t *testing.T) {
	t.Parallel()

	s, c, cleanup, err := miniredistest.New()
	assert.NilError(t, err)
	defer cleanup()

	db1, err := rdb.New(c, rdb.KeyPrefix("1"))
	assert.NilError(t, err)

	db2, err := rdb.New(c, rdb.KeyPrefix("2"))
	assert.NilError(t, err)

	seen, err := db1.Check("#foobar", "something")
	assert.NilError(t, err)
	assert.Assert(t, !seen)

	seen, err = db2.Check("#foobar", "something")
	assert.NilError(t, err)
	assert.Assert(t, !seen)

	s.FastForward(time.Second)

	assert.NilError(t, db1.Mark(10, "#foobar", "something"))

	s.FastForward(time.Second)

	seen, err = db1.Check("#foobar", "something")
	assert.NilError(t, err)
	assert.Assert(t, seen)

	seen, err = db2.Check("#foobar", "something")
	assert.NilError(t, err)
	assert.Assert(t, !seen)
}

func TestPrefixCollision(t *testing.T) {
	t.Parallel()

	s, c, cleanup, err := miniredistest.New()
	assert.NilError(t, err)
	defer cleanup()

	db1, err := rdb.New(c, rdb.KeyPrefix("prefix"))
	assert.NilError(t, err)

	db2, err := rdb.New(c)
	assert.NilError(t, err)

	seen, err := db1.Check("something")
	assert.NilError(t, err)
	assert.Assert(t, !seen)

	seen, err = db2.Check("prefix", "something")
	assert.NilError(t, err)
	assert.Assert(t, !seen)

	s.FastForward(time.Second)

	assert.NilError(t, db1.Mark(10, "something"))

	s.FastForward(time.Second)

	seen, err = db1.Check("something")
	assert.NilError(t, err)
	assert.Assert(t, seen)

	seen, err = db2.Check("prefix", "something")
	assert.NilError(t, err)
	assert.Assert(t, !seen)
}

func TestBadCheckAndMarkScript(t *testing.T) {
	defer rdb.ReplaceCheckAndMark("local")()

	_, c, cleanup, err := miniredistest.New()
	assert.NilError(t, err)
	defer cleanup()

	_, err = rdb.New(c)
	assert.ErrorContains(t, err, "syntax error")
}

func TestBadCheckAndRefreshScript(t *testing.T) {
	defer rdb.ReplaceCheckAndRefresh("local")()

	_, c, cleanup, err := miniredistest.New()
	assert.NilError(t, err)
	defer cleanup()

	_, err = rdb.New(c)
	assert.ErrorContains(t, err, "syntax error")
}
