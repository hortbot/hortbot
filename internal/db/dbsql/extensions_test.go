package dbsql_test

import (
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/hortbot/hortbot/internal/db/dbsql"
	"github.com/hortbot/hortbot/internal/db/migrations"
	"github.com/hortbot/hortbot/internal/pkg/testpostgres"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"gotest.tools/v3/assert"
)

func TestSaveQueryModelCoverage(t *testing.T) {
	t.Parallel()

	channelFields := jsonFields(
		reflect.TypeFor[dbsql.UpdateChannelMembershipParams](),
		reflect.TypeFor[dbsql.UpdateChannelUserListsParams](),
		reflect.TypeFor[dbsql.UpdateChannelRaffleEnabledParams](),
		reflect.TypeFor[dbsql.UpdateChannelSettingsParams](),
		reflect.TypeFor[dbsql.UpdateChannelActivityParams](),
	)
	expectedChannelFields := jsonFields(reflect.TypeFor[dbsql.Channel]())
	expectedChannelFields = removeFields(expectedChannelFields,
		"created_at",
		"updated_at",
		"twitch_id",
		// These legacy fields have no mutation surface.
		"sub_message",
		"sub_message_enabled",
		"resub_message",
		"resub_message_enabled",
	)
	assert.DeepEqual(t, channelFields, expectedChannelFields)

	tokenFields := jsonFields(reflect.TypeFor[dbsql.UpsertTwitchTokenParams]())
	expectedTokenFields := removeFields(jsonFields(reflect.TypeFor[dbsql.TwitchToken]()),
		"id",
		"created_at",
		"updated_at",
	)
	assert.DeepEqual(t, tokenFields, expectedTokenFields)
}

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

func jsonFields(types ...reflect.Type) []string {
	fields := make(map[string]struct{})
	for _, typ := range types {
		for field := range typ.Fields() {
			name := strings.Split(field.Tag.Get("json"), ",")[0]
			if name != "" && name != "-" {
				fields[name] = struct{}{}
			}
		}
	}

	names := make([]string, 0, len(fields))
	for name := range fields {
		names = append(names, name)
	}
	slices.Sort(names)
	return names
}

func removeFields(fields []string, removed ...string) []string {
	for _, field := range removed {
		fields = slices.DeleteFunc(fields, func(value string) bool {
			return value == field
		})
	}
	return fields
}
