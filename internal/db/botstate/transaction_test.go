package botstate_test

import (
	"context"
	"testing"
	"time"

	"gotest.tools/v3/assert"
)

func TestExecutorTransactionRollsBackStateOperations(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()

	state, sqlDB, _ := freshStoreAndDB(t)
	assert.NilError(t, state.RaffleAdd(ctx, "ch", "user"))

	tx, err := sqlDB.BeginTx(ctx, nil)
	assert.NilError(t, err)

	assert.NilError(t, state.Store.IncrementBuiltinUsageStat(ctx, tx, "test"))
	assert.NilError(t, state.Store.MarkCooldown(ctx, tx, "ch", "command", time.Hour))

	confirmed, err := state.Store.Confirm(ctx, tx, "ch", "user", "action", time.Hour)
	assert.NilError(t, err)
	assert.Assert(t, !confirmed)

	winners, err := state.Store.RaffleWinners(ctx, tx, "ch", 1)
	assert.NilError(t, err)
	assert.DeepEqual(t, winners, []string{"user"})

	assert.NilError(t, tx.Rollback())

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
