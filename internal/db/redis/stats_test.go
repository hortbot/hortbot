package redis_test

import (
	"context"
	"testing"
	"time"

	"github.com/hortbot/hortbot/internal/db/redis"
	"github.com/hortbot/hortbot/internal/pkg/testutil/miniredistest"
	"gotest.tools/v3/assert"
)

func TestBultinUsageStats(t *testing.T) {
	t.Parallel()

	_, c, cleanup, err := miniredistest.New()
	assert.NilError(t, err)
	defer cleanup()

	db := redis.New(c)

	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()

	stats, err := db.GetBuiltinUsageStats(ctx)
	assert.NilError(t, err)
	assert.Equal(t, len(stats), 0)

	err = db.IncrementBuiltinUsageStat(ctx, "command")
	assert.NilError(t, err)

	stats, err = db.GetBuiltinUsageStats(ctx)
	assert.NilError(t, err)
	assert.DeepEqual(t, stats, map[string]string{
		"command": "1",
	})

	err = db.IncrementBuiltinUsageStat(ctx, "command")
	assert.NilError(t, err)

	stats, err = db.GetBuiltinUsageStats(ctx)
	assert.NilError(t, err)
	assert.DeepEqual(t, stats, map[string]string{
		"command": "2",
	})
}

func TestActionUsageStats(t *testing.T) {
	t.Parallel()

	_, c, cleanup, err := miniredistest.New()
	assert.NilError(t, err)
	defer cleanup()

	db := redis.New(c)

	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()

	stats, err := db.GetActionUsageStats(ctx)
	assert.NilError(t, err)
	assert.Equal(t, len(stats), 0)

	err = db.IncrementActionUsageStat(ctx, "GAME")
	assert.NilError(t, err)

	stats, err = db.GetActionUsageStats(ctx)
	assert.NilError(t, err)
	assert.DeepEqual(t, stats, map[string]string{
		"GAME": "1",
	})

	err = db.IncrementActionUsageStat(ctx, "GAME")
	assert.NilError(t, err)

	stats, err = db.GetActionUsageStats(ctx)
	assert.NilError(t, err)
	assert.DeepEqual(t, stats, map[string]string{
		"GAME": "2",
	})
}
