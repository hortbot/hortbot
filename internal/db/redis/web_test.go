package redis_test

import (
	"context"
	"testing"
	"time"

	"github.com/hortbot/hortbot/internal/db/redis"
	"github.com/hortbot/hortbot/internal/pkg/testutil/miniredistest"
	"gotest.tools/v3/assert"
)

func TestAuthState(t *testing.T) {
	t.Parallel()

	s, c, cleanup, err := miniredistest.New()
	assert.NilError(t, err)
	defer cleanup()

	db := redis.New(c)

	const key = "some_state"

	type value struct {
		S string
		X int
		T time.Time
	}

	orig := &value{
		S: "string",
		X: 1234,
		T: time.Time{}.Add(time.Hour),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var got value
	ok, err := db.GetAuthState(ctx, key, &got)
	assert.NilError(t, err)
	assert.Equal(t, ok, false)

	err = db.SetAuthState(ctx, key, orig, time.Hour)
	assert.NilError(t, err)

	s.FastForward(time.Hour / 2)

	got = value{}
	ok, err = db.GetAuthState(ctx, key, &got)
	assert.NilError(t, err)
	assert.Equal(t, ok, true)
	assert.DeepEqual(t, &got, orig)

	got = value{}
	ok, err = db.GetAuthState(ctx, key, &got)
	assert.NilError(t, err)
	assert.Equal(t, ok, false)

	err = db.SetAuthState(ctx, key, orig, time.Hour)
	assert.NilError(t, err)

	s.FastForward(time.Hour * 2)

	got = value{}
	ok, err = db.GetAuthState(ctx, key, &got)
	assert.NilError(t, err)
	assert.Equal(t, ok, false)
}
