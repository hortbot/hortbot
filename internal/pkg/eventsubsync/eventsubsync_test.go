package eventsubsync_test

import (
	"testing"

	"github.com/hortbot/hortbot/internal/pkg/eventsubsync"
	"github.com/hortbot/hortbot/internal/pkg/testpostgres/pgpool"
	"gotest.tools/v3/assert"
)

var pool pgpool.Pool

func TestRequests(t *testing.T) {
	t.Parallel()

	db := pool.FreshDB(t)
	requests := eventsubsync.Requests{}

	version, err := requests.Version(t.Context(), db)
	assert.NilError(t, err)
	assert.Equal(t, version, int64(0))

	assert.NilError(t, requests.NotifyEventsubUpdates(t.Context(), db))
	assert.NilError(t, requests.NotifyEventsubUpdates(t.Context(), db))

	version, err = requests.Version(t.Context(), db)
	assert.NilError(t, err)
	assert.Equal(t, version, int64(2))

	tx, err := db.BeginTx(t.Context(), nil)
	assert.NilError(t, err)
	assert.NilError(t, requests.NotifyEventsubUpdates(t.Context(), tx))
	assert.NilError(t, tx.Rollback())

	version, err = requests.Version(t.Context(), db)
	assert.NilError(t, err)
	assert.Equal(t, version, int64(2))
}
