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

	const state = "some_state"

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	ok, err := db.CheckAuthState(ctx, state)
	assert.NilError(t, err)
	assert.Equal(t, ok, false)

	err = db.SetAuthState(ctx, state, time.Hour)
	assert.NilError(t, err)

	s.FastForward(time.Hour / 2)

	ok, err = db.CheckAuthState(ctx, state)
	assert.NilError(t, err)
	assert.Equal(t, ok, true)

	ok, err = db.CheckAuthState(ctx, state)
	assert.NilError(t, err)
	assert.Equal(t, ok, false)

	err = db.SetAuthState(ctx, state, time.Hour)
	assert.NilError(t, err)

	s.FastForward(time.Hour * 2)

	ok, err = db.CheckAuthState(ctx, state)
	assert.NilError(t, err)
	assert.Equal(t, ok, false)
}
