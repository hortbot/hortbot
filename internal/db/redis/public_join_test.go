package redis_test

import (
	"context"
	"testing"
	"time"

	"github.com/hortbot/hortbot/internal/db/redis"
	"github.com/hortbot/hortbot/internal/pkg/testutil/miniredistest"
	"gotest.tools/v3/assert"
)

func TestPublicJoin(t *testing.T) {
	t.Parallel()

	_, c, cleanup, err := miniredistest.New()
	assert.NilError(t, err)
	defer cleanup()

	db := redis.New(c)

	const channel = "channel"

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	v, err := db.PublicJoin(ctx, channel)
	assert.NilError(t, err)
	assert.Assert(t, v == nil)

	assert.NilError(t, db.SetPublicJoin(ctx, channel, true))

	v, err = db.PublicJoin(ctx, channel)
	assert.NilError(t, err)
	assert.Assert(t, v != nil)
	assert.Assert(t, *v)

	assert.NilError(t, db.SetPublicJoin(ctx, channel, false))

	v, err = db.PublicJoin(ctx, channel)
	assert.NilError(t, err)
	assert.Assert(t, v != nil)
	assert.Assert(t, !*v)

	assert.NilError(t, db.UnsetPublicJoin(ctx, channel))

	v, err = db.PublicJoin(ctx, channel)
	assert.NilError(t, err)
	assert.Assert(t, v == nil)

}
