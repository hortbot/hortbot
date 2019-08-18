package rdb_test

import (
	"testing"

	"github.com/hortbot/hortbot/internal/pkg/rdb"
	"github.com/hortbot/hortbot/internal/pkg/testutil/miniredistest"
	"gotest.tools/v3/assert"
)

func TestSetEmpty(t *testing.T) {
	t.Parallel()

	_, c, cleanup, err := miniredistest.New()
	assert.NilError(t, err)
	defer cleanup()

	db, err := rdb.New(c)
	assert.NilError(t, err)

	l, err := db.SetLen("foo", "bar")
	assert.NilError(t, err)
	assert.Equal(t, l, int64(0))

	s, ok, err := db.SetPop("foo", "bar")
	assert.NilError(t, err)
	assert.Equal(t, s, "")
	assert.Assert(t, !ok)

	assert.NilError(t, db.SetClear("foo", "bar"))

	l, err = db.SetLen("foo", "bar")
	assert.NilError(t, err)
	assert.Equal(t, l, int64(0))
}

func TestSetAdd(t *testing.T) {
	t.Parallel()

	_, c, cleanup, err := miniredistest.New()
	assert.NilError(t, err)
	defer cleanup()

	db, err := rdb.New(c)
	assert.NilError(t, err)

	l, err := db.SetLen("foobar")
	assert.NilError(t, err)
	assert.Equal(t, l, int64(0))

	assert.NilError(t, db.SetAdd("v1", "foobar"))

	l, err = db.SetLen("foobar")
	assert.NilError(t, err)
	assert.Equal(t, l, int64(1))

	assert.NilError(t, db.SetAdd("v2", "foobar"))

	l, err = db.SetLen("foobar")
	assert.NilError(t, err)
	assert.Equal(t, l, int64(2))

	assert.NilError(t, db.SetAdd("v2", "foobar"))

	l, err = db.SetLen("foobar")
	assert.NilError(t, err)
	assert.Equal(t, l, int64(2))
}

func TestSetClear(t *testing.T) {
	t.Parallel()

	_, c, cleanup, err := miniredistest.New()
	assert.NilError(t, err)
	defer cleanup()

	db, err := rdb.New(c)
	assert.NilError(t, err)

	l, err := db.SetLen("foobar")
	assert.NilError(t, err)
	assert.Equal(t, l, int64(0))

	assert.NilError(t, db.SetAdd("v1", "foobar"))
	assert.NilError(t, db.SetAdd("v2", "foobar"))
	assert.NilError(t, db.SetAdd("v3", "foobar"))

	l, err = db.SetLen("foobar")
	assert.NilError(t, err)
	assert.Equal(t, l, int64(3))

	assert.NilError(t, db.SetClear("foobar"))

	l, err = db.SetLen("foobar")
	assert.NilError(t, err)
	assert.Equal(t, l, int64(0))
}

func TestSetPop(t *testing.T) {
	t.Parallel()

	s, c, cleanup, err := miniredistest.New()
	assert.NilError(t, err)
	defer cleanup()

	s.Seed(311)

	db, err := rdb.New(c)
	assert.NilError(t, err)

	l, err := db.SetLen("foobar")
	assert.NilError(t, err)
	assert.Equal(t, l, int64(0))

	assert.NilError(t, db.SetAdd("v1", "foobar"))
	assert.NilError(t, db.SetAdd("v2", "foobar"))
	assert.NilError(t, db.SetAdd("v3", "foobar"))

	v, ok, err := db.SetPop("foobar")
	assert.NilError(t, err)
	assert.Assert(t, ok)
	assert.Equal(t, v, "v2")

	v, ok, err = db.SetPop("foobar")
	assert.NilError(t, err)
	assert.Assert(t, ok)
	assert.Equal(t, v, "v3")

	v, ok, err = db.SetPop("foobar")
	assert.NilError(t, err)
	assert.Assert(t, ok)
	assert.Equal(t, v, "v1")

	v, ok, err = db.SetPop("foobar")
	assert.NilError(t, err)
	assert.Assert(t, !ok)
	assert.Equal(t, v, "")
}
