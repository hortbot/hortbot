package dbsql_test

import (
	"testing"
	"time"

	"github.com/hortbot/hortbot/internal/db/dbsql"
	"github.com/hortbot/hortbot/internal/db/migrations"
	"github.com/hortbot/hortbot/internal/pkg/testpostgres"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"gotest.tools/v3/assert"
)

func TestDeleteChannelCascadeDeletesHighlights(t *testing.T) {
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
		Name:        "delete-highlights",
		DisplayName: "Delete Highlights",
		BotName:     "hortbot",
	})
	assert.NilError(t, err)
	assert.NilError(t, queries.InsertHighlight(t.Context(), dbsql.InsertHighlightParams{
		ChannelID:     channel.ID,
		HighlightedAt: dbsql.TimestamptzFrom(time.Now()),
		StartedAt:     pgtype.Timestamptz{},
		Status:        "complete",
		Game:          "Test Game",
	}))

	tx, err := db.Begin(t.Context())
	assert.NilError(t, err)
	txQueries := dbsql.New(tx)
	assert.NilError(t, txQueries.DeleteChannelCascade(t.Context(), channel.ID))
	assert.NilError(t, tx.Commit(t.Context()))

	_, err = queries.GetChannelByID(t.Context(), channel.ID)
	assert.ErrorIs(t, err, pgx.ErrNoRows)
	highlights, err := queries.ListHighlights(t.Context(), channel.ID)
	assert.NilError(t, err)
	assert.Equal(t, len(highlights), 0)
}
