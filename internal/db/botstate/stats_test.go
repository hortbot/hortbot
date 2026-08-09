package botstate_test

import (
	"context"
	"testing"
	"time"

	"gotest.tools/v3/assert"
)

func TestBuiltinUsageStats(t *testing.T) {
	t.Parallel()

	db, _ := freshStore(t)

	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()

	stats, err := db.GetBuiltinUsageStats(ctx)
	assert.NilError(t, err)
	assert.Equal(t, len(stats), 0)

	err = db.IncrementBuiltinUsageStat(ctx, "command")
	assert.NilError(t, err)

	stats, err = db.GetBuiltinUsageStats(ctx)
	assert.NilError(t, err)
	assert.DeepEqual(t, stats, map[string]string{"command": "1"})

	err = db.IncrementBuiltinUsageStat(ctx, "command")
	assert.NilError(t, err)

	stats, err = db.GetBuiltinUsageStats(ctx)
	assert.NilError(t, err)
	assert.DeepEqual(t, stats, map[string]string{"command": "2"})

	err = db.IncrementBuiltinUsageStat(ctx, "other")
	assert.NilError(t, err)

	stats, err = db.GetBuiltinUsageStats(ctx)
	assert.NilError(t, err)
	assert.DeepEqual(t, stats, map[string]string{"command": "2", "other": "1"})
}

func TestActionUsageStats(t *testing.T) {
	t.Parallel()

	db, _ := freshStore(t)

	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()

	stats, err := db.GetActionUsageStats(ctx)
	assert.NilError(t, err)
	assert.Equal(t, len(stats), 0)

	err = db.IncrementActionUsageStat(ctx, "GAME")
	assert.NilError(t, err)

	stats, err = db.GetActionUsageStats(ctx)
	assert.NilError(t, err)
	assert.DeepEqual(t, stats, map[string]string{"GAME": "1"})

	err = db.IncrementActionUsageStat(ctx, "GAME")
	assert.NilError(t, err)

	stats, err = db.GetActionUsageStats(ctx)
	assert.NilError(t, err)
	assert.DeepEqual(t, stats, map[string]string{"GAME": "2"})
}

func TestUsageStatsBucketsAreIsolated(t *testing.T) {
	t.Parallel()

	db, _ := freshStore(t)

	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()

	assert.NilError(t, db.IncrementBuiltinUsageStat(ctx, "name"))
	assert.NilError(t, db.IncrementActionUsageStat(ctx, "name"))
	assert.NilError(t, db.IncrementActionUsageStat(ctx, "name"))

	builtin, err := db.GetBuiltinUsageStats(ctx)
	assert.NilError(t, err)
	assert.DeepEqual(t, builtin, map[string]string{"name": "1"})

	action, err := db.GetActionUsageStats(ctx)
	assert.NilError(t, err)
	assert.DeepEqual(t, action, map[string]string{"name": "2"})
}
