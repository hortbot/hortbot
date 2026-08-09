package bot

import (
	"context"
	"database/sql"
	"errors"
	"maps"
	"slices"
	"testing"

	"github.com/hortbot/hortbot/internal/db/botstate"
	"github.com/hortbot/hortbot/internal/db/migrations"
	"github.com/hortbot/hortbot/internal/pkg/testpostgres"
	"gotest.tools/v3/assert"
)

func TestImportLegacyRedisState(t *testing.T) {
	t.Parallel()

	db := redisImportDB(t)
	state := botstate.New()
	assert.NilError(t, state.MergeUsageStats(t.Context(), db, map[string]int64{"existing": 10}, nil))
	assert.NilError(t, state.RaffleAdd(t.Context(), db, "channel", "existing"))

	source := &fakeLegacyRedis{
		hashes: map[string]map[string]string{
			legacyBuiltinUsageStats: {"command": "3", "existing": "5"},
			legacyActionUsageStats:  {"GAME": "7"},
		},
		sets: map[string][]string{
			"channel:channel:raffle:": {"alice", "bob", "existing"},
		},
	}

	imported, err := importLegacyRedisState(t.Context(), db, state, source)
	assert.NilError(t, err)
	assert.DeepEqual(t, imported, redisImportResult{
		BuiltinStats: 2,
		ActionStats:  1,
		Raffles:      1,
		RaffleUsers:  3,
	})

	builtins, err := state.GetBuiltinUsageStats(t.Context(), db)
	assert.NilError(t, err)
	assert.DeepEqual(t, builtins, map[string]string{"command": "3", "existing": "10"})
	actions, err := state.GetActionUsageStats(t.Context(), db)
	assert.NilError(t, err)
	assert.DeepEqual(t, actions, map[string]string{"GAME": "7"})

	rows, err := db.QueryContext(t.Context(), `
		SELECT user_id FROM bot_raffle_entries
		WHERE channel = 'channel'
		ORDER BY user_id
	`)
	assert.NilError(t, err)
	defer rows.Close()
	var users []string
	for rows.Next() {
		var user string
		assert.NilError(t, rows.Scan(&user))
		users = append(users, user)
	}
	assert.NilError(t, rows.Err())
	assert.DeepEqual(t, users, []string{"alice", "bob", "existing"})
	assert.Equal(t, len(source.hashes), 0)
	assert.Equal(t, len(source.sets), 0)

	imported, err = importLegacyRedisState(t.Context(), db, state, source)
	assert.NilError(t, err)
	assert.DeepEqual(t, imported, redisImportResult{})
}

func TestImportLegacyRedisStateRetriesAfterDeleteFailure(t *testing.T) {
	t.Parallel()

	db := redisImportDB(t)
	state := botstate.New()
	source := &fakeLegacyRedis{
		hashes: map[string]map[string]string{
			legacyBuiltinUsageStats: {"command": "3"},
		},
		deleteErr: errors.New("Redis unavailable"),
	}

	_, err := importLegacyRedisState(t.Context(), db, state, source)
	assert.ErrorContains(t, err, "delete imported Redis usage stats")

	source.deleteErr = nil
	_, err = importLegacyRedisState(t.Context(), db, state, source)
	assert.NilError(t, err)

	builtins, err := state.GetBuiltinUsageStats(t.Context(), db)
	assert.NilError(t, err)
	assert.DeepEqual(t, builtins, map[string]string{"command": "3"})
}

func redisImportDB(t *testing.T) *sql.DB {
	t.Helper()

	pdb, err := testpostgres.New()
	assert.NilError(t, err)
	t.Cleanup(pdb.Cleanup)
	assert.NilError(t, migrations.Up(pdb.ConnStr(), t.Logf))

	db, err := pdb.Open()
	assert.NilError(t, err)
	t.Cleanup(func() { assert.NilError(t, db.Close()) })
	return db
}

type fakeLegacyRedis struct {
	hashes    map[string]map[string]string
	sets      map[string][]string
	deleteErr error
}

func (f *fakeLegacyRedis) Hash(_ context.Context, key string) (map[string]string, error) {
	return maps.Clone(f.hashes[key]), nil
}

func (f *fakeLegacyRedis) SetMembers(_ context.Context, key string) ([]string, error) {
	return slices.Clone(f.sets[key]), nil
}

func (f *fakeLegacyRedis) Scan(_ context.Context, cursor uint64, _ string, _ int64) ([]string, uint64, error) {
	if cursor != 0 {
		return nil, 0, nil
	}
	return slices.Sorted(maps.Keys(f.sets)), 0, nil
}

func (f *fakeLegacyRedis) Delete(_ context.Context, keys ...string) error {
	if f.deleteErr != nil {
		return f.deleteErr
	}
	for _, key := range keys {
		delete(f.hashes, key)
		delete(f.sets, key)
	}
	return nil
}
