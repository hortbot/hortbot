package btest

import (
	"context"
	"encoding/json"
	"strconv"
	"testing"

	"github.com/hortbot/hortbot/internal/db/models"
	"github.com/hortbot/hortbot/internal/db/modelsx"
	"github.com/volatiletech/sqlboiler/v4/boil"
	"gotest.tools/v3/assert"
)

func (st *scriptTester) insertChannel(t testing.TB, _, args string, lineNum int) {
	channel := modelsx.NewChannel(0, "", "", "")
	assert.NilError(t, json.Unmarshal([]byte(args), channel), "line %d", lineNum)

	st.addAction(func(ctx context.Context) {
		assert.NilError(t, channel.Insert(ctx, st.db, boil.Infer()), "line %d", lineNum)
	})
}

func (st *scriptTester) insertCustomCommand(t testing.TB, _, args string, lineNum int) {
	var sc models.CustomCommand
	assert.NilError(t, json.Unmarshal([]byte(args), &sc), "line %d", lineNum)

	st.addAction(func(ctx context.Context) {
		assert.NilError(t, sc.Insert(ctx, st.db, boil.Infer()), "line %d", lineNum)
	})
}

func (st *scriptTester) insertRepeatedCommand(t testing.TB, _, args string, lineNum int) {
	var rc models.RepeatedCommand
	assert.NilError(t, json.Unmarshal([]byte(args), &rc), "line %d", lineNum)

	st.addAction(func(ctx context.Context) {
		ctx = boil.SkipTimestamps(ctx)
		assert.NilError(t, rc.Insert(ctx, st.db, boil.Infer()), "line %d", lineNum)
	})
}

func (st *scriptTester) insertScheduledCommand(t testing.TB, _, args string, lineNum int) {
	var sc models.ScheduledCommand
	assert.NilError(t, json.Unmarshal([]byte(args), &sc), "line %d", lineNum)

	st.addAction(func(ctx context.Context) {
		ctx = boil.SkipTimestamps(ctx)
		assert.NilError(t, sc.Insert(ctx, st.db, boil.Infer()), "line %d", lineNum)
	})
}

func (st *scriptTester) upsertTwitchToken(t testing.TB, _, args string, lineNum int) {
	var tt models.TwitchToken
	assert.NilError(t, json.Unmarshal([]byte(args), &tt), "line %d", lineNum)

	st.addAction(func(ctx context.Context) {
		if tt.BotName.Valid {
			st.idToName[tt.TwitchID] = tt.BotName.String
		}
		ctx = boil.SkipTimestamps(ctx)
		assert.NilError(t, modelsx.UpsertToken(ctx, st.db, &tt), "line %d", lineNum)
	})
}

func (st *scriptTester) deleteTwitchToken(t testing.TB, _, args string, lineNum int) {
	id, err := strconv.ParseInt(args, 10, 64)
	assert.NilError(t, err, "line %d", lineNum)

	st.addAction(func(ctx context.Context) {
		ctx = boil.SkipTimestamps(ctx)
		assert.NilError(t, models.TwitchTokens(models.TwitchTokenWhere.TwitchID.EQ(id)).DeleteAll(ctx, st.db), "line %d", lineNum)
	})
}

func (st *scriptTester) insertCommandInfo(t testing.TB, _, args string, lineNum int) {
	var ci models.CommandInfo
	assert.NilError(t, json.Unmarshal([]byte(args), &ci), "line %d", lineNum)

	st.addAction(func(ctx context.Context) {
		ctx = boil.SkipTimestamps(ctx)
		assert.NilError(t, ci.Insert(ctx, st.db, boil.Infer()), "line %d", lineNum)
	})
}
