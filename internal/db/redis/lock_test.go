package redis_test

import (
	"context"
	"testing"
	"time"

	"github.com/hortbot/hortbot/internal/db/redis"
	"github.com/hortbot/hortbot/internal/pkg/testutil/miniredistest"
	"gotest.tools/v3/assert"
)

func TestChannelLock(t *testing.T) {
	t.Parallel()

	_, c, cleanup, err := miniredistest.New()
	assert.NilError(t, err)
	defer cleanup()

	db := redis.New(c)

	const channel = "some_state"

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	const (
		ttl     = time.Second
		maxWait = 2 * time.Second
	)

	lock, ok, err := db.LockChannel(ctx, channel+"other", ttl, maxWait)
	assert.NilError(t, err)
	assert.Assert(t, ok)

	lock, ok, err = db.LockChannel(ctx, channel, ttl, maxWait)
	assert.NilError(t, err)
	assert.Assert(t, ok)

	ok, err = lock.Refresh(10 * time.Second)
	assert.NilError(t, err)
	assert.Assert(t, ok)

	_, ok, err = db.LockChannel(ctx, channel, ttl, maxWait)
	assert.NilError(t, err)
	assert.Assert(t, !ok)

	assert.NilError(t, lock.Unlock())

	lock, ok, err = db.LockChannel(ctx, channel, ttl, maxWait)
	assert.NilError(t, err)
	assert.Assert(t, ok)

	remaining, err := lock.TTL()
	assert.NilError(t, err)
	assert.Assert(t, remaining > time.Second/2)
}
