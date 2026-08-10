package botstate_test

import (
	"context"
	"math/rand/v2"
	"testing"
	"time"

	"github.com/hortbot/hortbot/internal/db/botstate"
	"github.com/hortbot/hortbot/internal/db/dbsql"
	"gotest.tools/v3/assert"
)

func TestRaffleEmpty(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()

	db, _ := freshStore(t)

	n, err := db.RaffleCount(ctx, "foo")
	assert.NilError(t, err)
	assert.Equal(t, n, int64(0))

	winner, ok, err := db.RaffleWinner(ctx, "foo")
	assert.NilError(t, err)
	assert.Equal(t, winner, "")
	assert.Assert(t, !ok)

	winners, err := db.RaffleWinners(ctx, "foo", 5)
	assert.NilError(t, err)
	assert.DeepEqual(t, winners, []string{})

	assert.NilError(t, db.RaffleReset(ctx, "foo"))

	n, err = db.RaffleCount(ctx, "foo")
	assert.NilError(t, err)
	assert.Equal(t, n, int64(0))
}

func TestRaffleAddIsIdempotent(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()

	db, _ := freshStore(t)

	assert.NilError(t, db.RaffleAdd(ctx, "foo", "user1"))
	assert.NilError(t, db.RaffleAdd(ctx, "foo", "user2"))
	assert.NilError(t, db.RaffleAdd(ctx, "foo", "user2"))

	n, err := db.RaffleCount(ctx, "foo")
	assert.NilError(t, err)
	assert.Equal(t, n, int64(2))
}

func TestRaffleResetClearsOnlyTargetChannel(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()

	db, _ := freshStore(t)

	assert.NilError(t, db.RaffleAdd(ctx, "a", "user1"))
	assert.NilError(t, db.RaffleAdd(ctx, "b", "user1"))

	assert.NilError(t, db.RaffleReset(ctx, "a"))

	na, err := db.RaffleCount(ctx, "a")
	assert.NilError(t, err)
	assert.Equal(t, na, int64(0))

	nb, err := db.RaffleCount(ctx, "b")
	assert.NilError(t, err)
	assert.Equal(t, nb, int64(1))
}

func TestRaffleWinnerReturnsAndRemovesEntries(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()

	rng := rand.New(rand.NewPCG(311, 311)) //nolint:gosec // deterministic seed for test reproducibility, not security-sensitive.
	db := freshStoreWithRand(t, randAdapter{rng})

	assert.NilError(t, db.RaffleAdd(ctx, "foo", "v1"))
	assert.NilError(t, db.RaffleAdd(ctx, "foo", "v2"))
	assert.NilError(t, db.RaffleAdd(ctx, "foo", "v3"))

	got := map[string]bool{}
	for range 3 {
		v, ok, err := db.RaffleWinner(ctx, "foo")
		assert.NilError(t, err)
		assert.Assert(t, ok)
		assert.Assert(t, !got[v], "duplicate pop: %s", v)
		got[v] = true
	}

	v, ok, err := db.RaffleWinner(ctx, "foo")
	assert.NilError(t, err)
	assert.Assert(t, !ok)
	assert.Equal(t, v, "")

	assert.DeepEqual(t, got, map[string]bool{"v1": true, "v2": true, "v3": true})
}

func TestRaffleWinnersBatch(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()

	rng := rand.New(rand.NewPCG(311, 311)) //nolint:gosec // deterministic seed for test reproducibility, not security-sensitive.
	db := freshStoreWithRand(t, randAdapter{rng})

	for _, u := range []string{"v1", "v2", "v3", "v4"} {
		assert.NilError(t, db.RaffleAdd(ctx, "foo", u))
	}

	first, err := db.RaffleWinners(ctx, "foo", 3)
	assert.NilError(t, err)
	assert.Equal(t, len(first), 3)

	second, err := db.RaffleWinners(ctx, "foo", 2)
	assert.NilError(t, err)
	assert.Equal(t, len(second), 1, "only one entry remains after popping 3 of 4")

	third, err := db.RaffleWinners(ctx, "foo", 2)
	assert.NilError(t, err)
	assert.DeepEqual(t, third, []string{})

	all := map[string]bool{}
	for _, v := range first {
		all[v] = true
	}
	for _, v := range second {
		all[v] = true
	}
	assert.DeepEqual(t, all, map[string]bool{"v1": true, "v2": true, "v3": true, "v4": true})
}

func TestRaffleWinnersNonPositive(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()

	db, _ := freshStore(t)

	assert.NilError(t, db.RaffleAdd(ctx, "foo", "user1"))

	for _, n := range []int64{0, -1, -100} {
		got, err := db.RaffleWinners(ctx, "foo", n)
		assert.NilError(t, err)
		assert.DeepEqual(t, got, []string{})
	}

	count, err := db.RaffleCount(ctx, "foo")
	assert.NilError(t, err)
	assert.Equal(t, count, int64(1), "non-positive n must not consume entries")
}

func freshStoreWithRand(t testing.TB, rng botstate.Rand) *testStore {
	t.Helper()
	sqlDB := pool.FreshDB(t)
	t.Cleanup(func() { sqlDB.Close() })
	return &testStore{
		Store:   botstate.New(botstate.WithRand(rng)),
		db:      sqlDB,
		queries: dbsql.New(sqlDB),
	}
}

type randAdapter struct{ *rand.Rand }

func (r randAdapter) IntN(n int) int { return r.Rand.IntN(n) }
