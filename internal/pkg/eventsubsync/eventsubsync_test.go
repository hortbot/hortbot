package eventsubsync_test

import (
	"testing"

	"github.com/hortbot/hortbot/internal/db/dbsql"
	"github.com/hortbot/hortbot/internal/pkg/eventsubsync"
	"github.com/hortbot/hortbot/internal/pkg/testpostgres/pgpool"
	"gotest.tools/v3/assert"
)

var pool pgpool.Pool

func TestRequests(t *testing.T) {
	t.Parallel()

	db := pool.FreshDB(t)
	queries := dbsql.New(db)
	requests := eventsubsync.Requests{}

	version, err := requests.Version(t.Context(), queries)
	assert.NilError(t, err)
	assert.Equal(t, version, int64(0))

	assert.NilError(t, requests.NotifyEventsubUpdates(t.Context(), queries))
	assert.NilError(t, requests.NotifyEventsubUpdates(t.Context(), queries))

	version, err = requests.Version(t.Context(), queries)
	assert.NilError(t, err)
	assert.Equal(t, version, int64(2))

	tx, err := db.Begin(t.Context())
	assert.NilError(t, err)
	assert.NilError(t, requests.NotifyEventsubUpdates(t.Context(), dbsql.New(tx)))
	assert.NilError(t, tx.Rollback(t.Context()))

	version, err = requests.Version(t.Context(), queries)
	assert.NilError(t, err)
	assert.Equal(t, version, int64(2))
}
