package botstate_test

import (
	"context"
	"testing"
	"time"

	"github.com/hortbot/hortbot/internal/db/dbsql"
	"gotest.tools/v3/assert"
)

func TestExecutorTransactionRollsBackStateOperations(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()

	state, sqlDB, _ := freshStoreAndDB(t)
	assert.NilError(t, state.RaffleAdd(ctx, "ch", "user"))

	tx, err := sqlDB.Begin(ctx)
	assert.NilError(t, err)
	queries := dbsql.New(tx)

	assert.NilError(t, state.Store.IncrementBuiltinUsageStat(ctx, queries, "test"))
	assert.NilError(t, state.Store.MarkCooldown(ctx, queries, "ch", "command", time.Hour))

	confirmed, err := state.Store.Confirm(ctx, queries, "ch", "user", "action", time.Hour)
	assert.NilError(t, err)
	assert.Assert(t, !confirmed)

	winners, err := state.Store.RaffleWinners(ctx, queries, "ch", 1)
	assert.NilError(t, err)
	assert.DeepEqual(t, winners, []string{"user"})

	assert.NilError(t, tx.Rollback(ctx))

	stats, err := state.GetBuiltinUsageStats(ctx)
	assert.NilError(t, err)
	assert.Equal(t, len(stats), 0)

	seen, err := state.CheckAndMarkCooldown(ctx, "ch", "command", time.Hour)
	assert.NilError(t, err)
	assert.Assert(t, !seen)

	confirmed, err = state.Confirm(ctx, "ch", "user", "action", time.Hour)
	assert.NilError(t, err)
	assert.Assert(t, !confirmed)

	count, err := state.RaffleCount(ctx, "ch")
	assert.NilError(t, err)
	assert.Equal(t, count, int64(1))
}
