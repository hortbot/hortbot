package redis_test

import (
	"context"
	"testing"
	"time"

	"github.com/hortbot/hortbot/internal/db/redis"
	"github.com/hortbot/hortbot/internal/pkg/testutil/miniredistest"
	"gotest.tools/v3/assert"
)

func TestDedupe(t *testing.T) {
	t.Parallel()

	_, c, cleanup, err := miniredistest.New()
	assert.NilError(t, err)
	defer cleanup()

	db := redis.New(c)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// All of these are tested in other packages.
	// Just verify that they don't crash.

	err = db.DedupeMark(ctx, "foo", time.Minute)
	assert.NilError(t, err)

	_, err = db.DedupeCheck(ctx, "foo", time.Minute)
	assert.NilError(t, err)

	_, err = db.DedupeCheckAndMark(ctx, "foo", time.Minute)
	assert.NilError(t, err)
}
