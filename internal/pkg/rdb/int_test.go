package rdb_test

import (
	"testing"
	"time"

	"github.com/hortbot/hortbot/internal/pkg/rdb"
	"github.com/hortbot/hortbot/internal/pkg/testutil/miniredistest"
	"gotest.tools/v3/assert"
)

func TestIncrementInt64(t *testing.T) {
	t.Parallel()

	s, c, cleanup, err := miniredistest.New()
	assert.NilError(t, err)
	defer cleanup()

	db, err := rdb.New(c)
	assert.NilError(t, err)

	n, err := db.Increment("#foobar", "something")
	assert.NilError(t, err)
	assert.Assert(t, n == 1)

	s.FastForward(time.Second)

	n, err = db.Increment("#foobar", "something")
	assert.NilError(t, err)
	assert.Assert(t, n == 2)

	n, err = db.GetInt64("#foobar", "something")
	assert.NilError(t, err)
	assert.Assert(t, n == 2)

	s.FastForward(time.Second)

	n, err = db.Increment("#foobar", "something_else")
	assert.NilError(t, err)
	assert.Assert(t, n == 1)

	n, err = db.GetInt64("#foobar", "something_else")
	assert.NilError(t, err)
	assert.Assert(t, n == 1)

	s.FastForward(time.Second)

	n, err = db.Increment("#foobar", "something")
	assert.NilError(t, err)
	assert.Assert(t, n == 3)

	n, err = db.GetInt64("#foobar", "something")
	assert.NilError(t, err)
	assert.Assert(t, n == 3)
}
