package botstate_test

import (
	"context"
	"testing"
	"time"

	"github.com/hortbot/hortbot/internal/db/dbsql"
	"gotest.tools/v3/assert"
)

func TestCleanupReturnsDatabaseErrors(t *testing.T) {
	t.Parallel()
	db, sqlDB, _ := freshStoreAndDB(t)
	sqlDB.Close()

	err := db.Cleanup(t.Context(), dbsql.New(sqlDB))
	assert.ErrorContains(t, err, "delete expired rows")
}

func TestCleanupDeletesExpiredRows(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
	defer cancel()

	state, sqlDB, _ := freshStoreAndDB(t)

	_, err := sqlDB.Exec(ctx, `
		INSERT INTO bot_command_cooldowns (channel, command_key, expires_at) VALUES
			('cleanup', 'stale', now() - interval '1 hour'),
			('cleanup', 'live',  now() + interval '1 hour')
	`)
	assert.NilError(t, err)

	assert.NilError(t, state.Cleanup(ctx, dbsql.New(sqlDB)))

	var staleCount, liveCount int
	assert.NilError(t, sqlDB.QueryRow(ctx,
		`SELECT count(*) FROM bot_command_cooldowns WHERE channel = 'cleanup' AND command_key = 'stale'`).Scan(&staleCount))
	assert.NilError(t, sqlDB.QueryRow(ctx,
		`SELECT count(*) FROM bot_command_cooldowns WHERE channel = 'cleanup' AND command_key = 'live'`).Scan(&liveCount))

	assert.Equal(t, staleCount, 0, "cleanup should have removed the stale row")
	assert.Equal(t, liveCount, 1, "cleanup must not touch live rows")
}
