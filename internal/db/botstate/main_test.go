//nolint:wrapcheck // testStore intentionally preserves the production API's errors for assertions.
package botstate_test

import (
	"context"
	"os"
	"sync/atomic"
	"testing"
	"time"

	"github.com/hortbot/hortbot/internal/db/botstate"
	"github.com/hortbot/hortbot/internal/db/dbsql"
	"github.com/hortbot/hortbot/internal/pkg/dbx"
	"github.com/hortbot/hortbot/internal/pkg/testpostgres/pgpool"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var pool pgpool.Pool

func TestMain(m *testing.M) {
	status := 1
	defer func() {
		if r := recover(); r != nil {
			panic(r)
		}
		os.Exit(status)
	}()
	defer pool.Cleanup()
	status = m.Run()
}

type fakeClock struct {
	base   time.Time
	offset atomic.Int64 // nanoseconds added to base
}

func newFakeClock(t time.Time) *fakeClock {
	return &fakeClock{base: t}
}

func (c *fakeClock) Now() time.Time {
	return c.base.Add(time.Duration(c.offset.Load()))
}

func (c *fakeClock) Advance(d time.Duration) {
	c.offset.Add(int64(d))
}

func freshStore(t testing.TB) (*testStore, *fakeClock) {
	state, _, clk := freshStoreAndDB(t)
	return state, clk
}

func freshStoreAndDB(t testing.TB) (*testStore, *pgxpool.Pool, *fakeClock) {
	t.Helper()
	db := pool.FreshDB(t)
	t.Cleanup(func() { db.Close() })

	clk := newFakeClock(time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC))
	state := &testStore{
		Store:   botstate.New(botstate.WithNow(clk.Now)),
		db:      db,
		queries: dbsql.New(db),
	}
	return state, db, clk
}

type testStore struct {
	*botstate.Store
	db      *pgxpool.Pool
	queries *dbsql.Queries
}

func transactValue[T any](ctx context.Context, db *pgxpool.Pool, fn func(*dbsql.Queries) (T, error)) (T, error) {
	var value T
	err := dbx.Transact(ctx, db, func(ctx context.Context, tx pgx.Tx) error {
		var err error
		value, err = fn(dbsql.New(tx))
		return err
	})
	return value, err
}

func (db *testStore) MarkCooldown(ctx context.Context, channel, key string, expiry time.Duration) error {
	return db.Store.MarkCooldown(ctx, db.queries, channel, key, expiry)
}

func (db *testStore) CheckAndMarkCooldown(ctx context.Context, channel, key string, expiry time.Duration) (bool, error) {
	return db.Store.CheckAndMarkCooldown(ctx, db.queries, channel, key, expiry)
}

func (db *testStore) RepeatAllowed(ctx context.Context, channel string, id int64, expiry time.Duration) (bool, error) {
	return db.Store.RepeatAllowed(ctx, db.queries, channel, id, expiry)
}

func (db *testStore) ScheduledAllowed(ctx context.Context, channel string, id int64, expiry time.Duration) (bool, error) {
	return db.Store.ScheduledAllowed(ctx, db.queries, channel, id, expiry)
}

func (db *testStore) AutoreplyAllowed(ctx context.Context, channel string, id int64, expiry time.Duration) (bool, error) {
	return db.Store.AutoreplyAllowed(ctx, db.queries, channel, id, expiry)
}

func (db *testStore) LinkPermit(ctx context.Context, channel, user string, expiry time.Duration) error {
	return db.Store.LinkPermit(ctx, db.queries, channel, user, expiry)
}

func (db *testStore) HasLinkPermit(ctx context.Context, channel, user string) (bool, error) {
	return db.Store.HasLinkPermit(ctx, db.queries, channel, user)
}

func (db *testStore) Confirm(ctx context.Context, channel, user, key string, expiry time.Duration) (bool, error) {
	return transactValue(ctx, db.db, func(queries *dbsql.Queries) (bool, error) {
		return db.Store.Confirm(ctx, queries, channel, user, key, expiry)
	})
}

func (db *testStore) FilterWarned(ctx context.Context, channel, user, filter string, expiry time.Duration) (bool, error) {
	return transactValue(ctx, db.db, func(queries *dbsql.Queries) (bool, error) {
		return db.Store.FilterWarned(ctx, queries, channel, user, filter, expiry)
	})
}

func (db *testStore) RaffleAdd(ctx context.Context, channel, user string) error {
	return db.Store.RaffleAdd(ctx, db.queries, channel, user)
}

func (db *testStore) RaffleReset(ctx context.Context, channel string) error {
	return db.Store.RaffleReset(ctx, db.queries, channel)
}

func (db *testStore) RaffleCount(ctx context.Context, channel string) (int64, error) {
	return db.Store.RaffleCount(ctx, db.queries, channel)
}

func (db *testStore) RaffleWinner(ctx context.Context, channel string) (string, bool, error) {
	type result struct {
		winner string
		ok     bool
	}
	value, err := transactValue(ctx, db.db, func(queries *dbsql.Queries) (result, error) {
		winner, ok, err := db.Store.RaffleWinner(ctx, queries, channel)
		return result{winner: winner, ok: ok}, err
	})
	return value.winner, value.ok, err
}

func (db *testStore) RaffleWinners(ctx context.Context, channel string, n int64) ([]string, error) {
	return transactValue(ctx, db.db, func(queries *dbsql.Queries) ([]string, error) {
		return db.Store.RaffleWinners(ctx, queries, channel, n)
	})
}

func (db *testStore) IncrementBuiltinUsageStat(ctx context.Context, name string) error {
	return db.Store.IncrementBuiltinUsageStat(ctx, db.queries, name)
}

func (db *testStore) GetBuiltinUsageStats(ctx context.Context) (map[string]string, error) {
	return db.Store.GetBuiltinUsageStats(ctx, db.queries)
}

func (db *testStore) IncrementActionUsageStat(ctx context.Context, name string) error {
	return db.Store.IncrementActionUsageStat(ctx, db.queries, name)
}

func (db *testStore) GetActionUsageStats(ctx context.Context) (map[string]string, error) {
	return db.Store.GetActionUsageStats(ctx, db.queries)
}

func (db *testStore) SetAuthState(ctx context.Context, key string, value any, expiry time.Duration) error {
	return db.Store.SetAuthState(ctx, db.queries, key, value, expiry)
}

func (db *testStore) GetAuthState(ctx context.Context, key string, value any) (bool, error) {
	return db.Store.GetAuthState(ctx, db.queries, key, value)
}
