package pgpool_test

import (
	"testing"

	"github.com/hortbot/hortbot/internal/db/models"
	"github.com/hortbot/hortbot/internal/pkg/docker/dpostgres/pgpool"
	"gotest.tools/v3/assert"
)

func TestPool(t *testing.T) {
	t.Parallel()
	var pool pgpool.Pool
	t.Cleanup(pool.Cleanup)

	db := pool.FreshDB(t)
	assert.Assert(t, db != nil)
	defer db.Close()

	count, err := models.Channels().Count(t.Context(), db)
	assert.NilError(t, err)
	assert.Equal(t, count, int64(0))
}

func TestPoolNoUse(t *testing.T) {
	t.Parallel()
	var pool pgpool.Pool
	pool.Cleanup()
}
