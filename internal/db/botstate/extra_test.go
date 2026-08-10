package botstate_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/hortbot/hortbot/internal/db/botstate"
	"github.com/hortbot/hortbot/internal/db/dbsql"
	"gotest.tools/v3/assert"
)

func TestDefaultRand(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()

	sqlDB := pool.FreshDB(t)
	db := &testStore{Store: botstate.New(), db: sqlDB, queries: dbsql.New(sqlDB)}

	assert.NilError(t, db.RaffleAdd(ctx, "default-rand", "winner"))
	winner, ok, err := db.RaffleWinner(ctx, "default-rand")
	assert.NilError(t, err)
	assert.Assert(t, ok)
	assert.Equal(t, winner, "winner")
}

func TestKeysWithColonsRoundTrip(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()

	sqlDB := pool.FreshDB(t)
	db := &testStore{Store: botstate.New(), db: sqlDB, queries: dbsql.New(sqlDB)}

	assert.NilError(t, db.LinkPermit(ctx, "channel:1", "evil:user", time.Minute))
	ok, err := db.HasLinkPermit(ctx, "channel:1", "evil:user")
	assert.NilError(t, err)
	assert.Assert(t, ok)

	assert.NilError(t, db.LinkPermit(ctx, "channel", "1:evil:user", time.Minute))
	ok, err = db.HasLinkPermit(ctx, "channel:1", "evil:user")
	assert.NilError(t, err)
	assert.Assert(t, !ok, "first permit was already consumed")

	ok, err = db.HasLinkPermit(ctx, "channel", "1:evil:user")
	assert.NilError(t, err)
	assert.Assert(t, ok, "second permit must be independent of the first")
}

func TestDump(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()

	state, sqlDB, _ := freshStoreAndDB(t)
	assert.NilError(t, state.MarkCooldown(ctx, "channel", "command", time.Minute))

	dump, err := state.Dump(ctx, dbsql.New(sqlDB))
	assert.NilError(t, err)
	assert.Assert(t, strings.Contains(dump, "bot_command_cooldowns\tchannel/command\t"))
}
