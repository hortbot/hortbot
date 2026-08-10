package dbsql_test

import (
	"testing"

	"github.com/hortbot/hortbot/internal/db/dbsql"
	"github.com/hortbot/hortbot/internal/db/migrations"
	"github.com/hortbot/hortbot/internal/pkg/testpostgres"
	"github.com/jackc/pgx/v5/pgtype"
	"gotest.tools/v3/assert"
)

func TestCompactAutorepliesUsesExistingNumber(t *testing.T) {
	t.Parallel()

	pdb, err := testpostgres.New()
	assert.NilError(t, err)
	t.Cleanup(pdb.Cleanup)
	assert.NilError(t, migrations.Up(pdb.ConnStr(), nil))

	db, err := pdb.Open(t.Context())
	assert.NilError(t, err)
	t.Cleanup(db.Close)

	queries := dbsql.New(db)
	channel, err := queries.InsertDefaultChannel(t.Context(), dbsql.InsertDefaultChannelParams{
		TwitchID:    1,
		Name:        "compact-autoreplies",
		DisplayName: "Compact Autoreplies",
		BotName:     "hortbot",
	})
	assert.NilError(t, err)

	for _, num := range []int32{1, 2, 4} {
		assert.NilError(t, queries.InsertAutoreply(t.Context(), dbsql.InsertAutoreplyParams{
			ChannelID:   channel.ID,
			Num:         num,
			Trigger:     "trigger",
			OrigPattern: pgtype.Text{},
			Response:    "response",
			Creator:     "tester",
			Editor:      "tester",
		}))
	}

	affected, err := queries.CompactAutoreplies(t.Context(), dbsql.CompactAutorepliesParams{
		StartNum:  1,
		ChannelID: channel.ID,
	})
	assert.NilError(t, err)
	assert.Equal(t, affected, int64(1))

	autoreplies, err := queries.ListAutoreplies(t.Context(), channel.ID)
	assert.NilError(t, err)
	assert.Equal(t, len(autoreplies), 3)
	assert.Equal(t, autoreplies[0].Num, int32(1))
	assert.Equal(t, autoreplies[1].Num, int32(2))
	assert.Equal(t, autoreplies[2].Num, int32(3))
}
