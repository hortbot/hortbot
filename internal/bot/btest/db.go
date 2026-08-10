package btest

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/hortbot/hortbot/internal/bot"
	"github.com/hortbot/hortbot/internal/confimport"
	"github.com/hortbot/hortbot/internal/db/dbsql"
	"github.com/jackc/pgx/v5/pgtype"
	"gotest.tools/v3/assert"
)

func (st *scriptTester) insertChannel(t testing.TB, _, args string, lineNum int) {
	channel := newFixtureChannel()
	assert.NilError(t, json.Unmarshal([]byte(args), channel), "line %d", lineNum)

	st.addAction(func(ctx context.Context) {
		assert.NilError(t, insertFixture(ctx, channel.ID, channel,
			st.queries.InsertFixtureChannel, st.queries.InsertImportedChannel), "line %d", lineNum)
	})
}

// Fixtures are partial JSON objects, so start them with the same defaults as
// the production InsertChannel query before applying the supplied fields.
func newFixtureChannel() *confimport.Channel {
	return &confimport.Channel{
		Active:                  true,
		Prefix:                  bot.DefaultChannelPrefix,
		Mode:                    dbsql.AccessLevelEveryone,
		ShouldModerate:          true,
		EnableWarnings:          true,
		SubsMayLink:             true,
		TimeoutDuration:         600,
		RollLevel:               dbsql.AccessLevelSubscriber,
		RollCooldown:            10,
		RollDefault:             20,
		FilterCapsPercentage:    50,
		FilterCapsMinCaps:       6,
		FilterSymbolsPercentage: 50,
		FilterSymbolsMinSymbols: 5,
		FilterMaxLength:         500,
		FilterEmotesMax:         4,
		Tweet:                   "Check out (_CHANNEL_URL_) playing (_GAME_) on @Twitch!",
		FilterExemptLevel:       dbsql.AccessLevelSubscriber,
	}
}

func (st *scriptTester) insertCustomCommand(t testing.TB, _, args string, lineNum int) {
	var sc confimport.CustomCommand
	assert.NilError(t, json.Unmarshal([]byte(args), &sc), "line %d", lineNum)

	st.addAction(func(ctx context.Context) {
		assert.NilError(t, insertFixture(ctx, sc.ID, &sc,
			st.queries.InsertFixtureCustomCommand, st.queries.InsertImportedCustomCommand), "line %d", lineNum)
	})
}

func (st *scriptTester) insertRepeatedCommand(t testing.TB, _, args string, lineNum int) {
	var rc confimport.RepeatedCommand
	assert.NilError(t, json.Unmarshal([]byte(args), &rc), "line %d", lineNum)

	st.addAction(func(ctx context.Context) {
		assert.NilError(t, insertFixture(ctx, rc.ID, &rc,
			st.queries.InsertFixtureRepeatedCommand, st.queries.InsertImportedRepeatedCommand), "line %d", lineNum)
	})
}

func (st *scriptTester) insertScheduledCommand(t testing.TB, _, args string, lineNum int) {
	var sc confimport.ScheduledCommand
	assert.NilError(t, json.Unmarshal([]byte(args), &sc), "line %d", lineNum)

	st.addAction(func(ctx context.Context) {
		assert.NilError(t, insertFixture(ctx, sc.ID, &sc,
			st.queries.InsertFixtureScheduledCommand, st.queries.InsertImportedScheduledCommand), "line %d", lineNum)
	})
}

func (st *scriptTester) upsertTwitchToken(t testing.TB, _, args string, lineNum int) {
	var tt dbsql.TwitchToken
	assert.NilError(t, json.Unmarshal([]byte(args), &tt), "line %d", lineNum)

	st.addAction(func(ctx context.Context) {
		if tt.BotName.Valid {
			st.idToName[tt.TwitchID] = tt.BotName.String
		}
		assert.NilError(t, st.queries.SaveTwitchToken(ctx, &tt), "line %d", lineNum)
	})
}

func (st *scriptTester) deleteTwitchToken(t testing.TB, _, args string, lineNum int) {
	id, err := strconv.ParseInt(args, 10, 64)
	assert.NilError(t, err, "line %d", lineNum)

	st.addAction(func(ctx context.Context) {
		assert.NilError(t, st.queries.DeleteTwitchTokenByID(ctx, id), "line %d", lineNum)
	})
}

func (st *scriptTester) insertCommandInfo(t testing.TB, _, args string, lineNum int) {
	var ci confimport.CommandInfo
	assert.NilError(t, json.Unmarshal([]byte(args), &ci), "line %d", lineNum)

	st.addAction(func(ctx context.Context) {
		assert.NilError(t, insertFixture(ctx, ci.ID, &ci,
			st.queries.InsertFixtureCommandInfo, st.queries.InsertImportedCommandInfo), "line %d", lineNum)
	})
}

type fixtureInserter func(context.Context, []byte) (int64, error)

func insertFixture(ctx context.Context, id int64, model any, withID, generated fixtureInserter) error {
	applyFixtureDefaults(model)
	data, err := json.Marshal(model)
	if err != nil {
		return fmt.Errorf("marshaling fixture: %w", err)
	}

	if id != 0 {
		_, err = withID(ctx, data)
	} else {
		_, err = generated(ctx, data)
	}
	return err
}

func applyFixtureDefaults(model any) {
	now := dbsql.TimestamptzFrom(time.Now())
	setTimestamps := func(createdAt, updatedAt *pgtype.Timestamptz) {
		if !createdAt.Valid {
			*createdAt = now
		}
		if !updatedAt.Valid {
			*updatedAt = now
		}
	}

	switch model := model.(type) {
	case *confimport.Channel:
		setTimestamps(&model.CreatedAt, &model.UpdatedAt)
		if !model.LastSeen.Valid {
			model.LastSeen = now
		}
		if model.Ignored == nil {
			model.Ignored = []string{}
		}
		if model.CustomOwners == nil {
			model.CustomOwners = []string{}
		}
		if model.CustomMods == nil {
			model.CustomMods = []string{}
		}
		if model.CustomRegulars == nil {
			model.CustomRegulars = []string{}
		}
		if model.PermittedLinks == nil {
			model.PermittedLinks = []string{}
		}
		if model.FilterBannedPhrasesPatterns == nil {
			model.FilterBannedPhrasesPatterns = []string{}
		}
	case *confimport.CustomCommand:
		setTimestamps(&model.CreatedAt, &model.UpdatedAt)
	case *confimport.CommandInfo:
		setTimestamps(&model.CreatedAt, &model.UpdatedAt)
	case *confimport.RepeatedCommand:
		setTimestamps(&model.CreatedAt, &model.UpdatedAt)
	case *confimport.ScheduledCommand:
		setTimestamps(&model.CreatedAt, &model.UpdatedAt)
	}
}
