package redis_test

import (
	"context"
	"testing"
	"time"

	"github.com/hortbot/hortbot/internal/db/redis"
	"github.com/hortbot/hortbot/internal/pkg/testutil/miniredistest"
	"gotest.tools/v3/assert"
)

func TestUsageStatistics(t *testing.T) {
	t.Parallel()

	_, c, cleanup, err := miniredistest.New()
	assert.NilError(t, err)
	defer cleanup()

	db := redis.New(c)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	stats, err := db.GetUsageStatistics(ctx)
	assert.NilError(t, err)
	assert.Equal(t, len(stats), 0)

	err = db.IncrementUsageStatistic(ctx, "command")
	assert.NilError(t, err)

	stats, err = db.GetUsageStatistics(ctx)
	assert.NilError(t, err)
	assert.DeepEqual(t, stats, map[string]string{
		"command": "1",
	})

	err = db.IncrementUsageStatistic(ctx, "command")
	assert.NilError(t, err)

	stats, err = db.GetUsageStatistics(ctx)
	assert.NilError(t, err)
	assert.DeepEqual(t, stats, map[string]string{
		"command": "2",
	})
}
