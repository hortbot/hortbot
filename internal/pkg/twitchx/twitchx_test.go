package twitchx_test

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	gocmp "github.com/google/go-cmp/cmp"
	"github.com/hortbot/hortbot/internal/db/models"
	"github.com/hortbot/hortbot/internal/db/modelsx"
	"github.com/hortbot/hortbot/internal/pkg/apiclient/twitch"
	"github.com/hortbot/hortbot/internal/pkg/apiclient/twitch/twitchfakes"
	"github.com/hortbot/hortbot/internal/pkg/oauth2x"
	"github.com/hortbot/hortbot/internal/pkg/testutil"
	"github.com/hortbot/hortbot/internal/pkg/twitchx"
	"github.com/volatiletech/null/v8"
	"github.com/volatiletech/sqlboiler/v4/boil"
	"github.com/zikaeroh/ctxlog"
	"golang.org/x/oauth2"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"
)

var (
	errTest  = errors.New("testing error")
	tokenCmp = gocmp.Comparer(func(x, y oauth2.Token) bool {
		return oauth2x.Equals(&x, &y)
	})
)

func TestFindBotTokenNoBot(t *testing.T) {
	t.Parallel()
	ctx, cancel := testContext(t)
	defer cancel()

	tw := &twitchfakes.FakeAPI{}
	db := pool.FreshDB(t)
	defer db.Close()

	tok, err := twitchx.FindBotToken(ctx, db, tw, "hortbot")
	assert.NilError(t, err)
	assert.Assert(t, tok == nil)
}

func TestFindBotToken(t *testing.T) {
	t.Parallel()
	ctx, cancel := testContext(t)
	defer cancel()

	tw := &twitchfakes.FakeAPI{}
	db := pool.FreshDB(t)
	defer db.Close()

	orig := &models.TwitchToken{
		TwitchID:     1234,
		BotName:      null.StringFrom("hortbot"),
		AccessToken:  "at-qwertyuiop1234567890",
		TokenType:    "Bearer",
		RefreshToken: "rt-qwertyuiop1234567890",
		Expiry:       time.Now(),
	}

	origTok := modelsx.ModelToToken(orig)

	assert.NilError(t, orig.Insert(ctx, db, boil.Infer()))

	tok, err := twitchx.FindBotToken(ctx, db, tw, orig.BotName.String)
	assert.NilError(t, err)

	assert.Assert(t, cmp.DeepEqual(origTok, tok, tokenCmp))
}

func TestFindBotTokenGetError(t *testing.T) {
	t.Parallel()
	ctx, cancel := testContext(t)
	defer cancel()

	tw := &twitchfakes.FakeAPI{
		GetUserForTokenStub: func(context.Context, *oauth2.Token) (*twitch.User, *oauth2.Token, error) {
			return nil, nil, errTest
		},
	}
	db := pool.FreshDB(t)
	defer db.Close()

	orig := &models.TwitchToken{
		TwitchID:     1234,
		BotName:      null.StringFrom("hortbot"),
		AccessToken:  "at-qwertyuiop1234567890",
		TokenType:    "Bearer",
		RefreshToken: "rt-qwertyuiop1234567890",
		Expiry:       time.Now(),
	}

	assert.NilError(t, orig.Insert(ctx, db, boil.Infer()))

	_, err := twitchx.FindBotToken(ctx, db, tw, "hortbot")
	assert.Equal(t, err, errTest)
}

func TestFindBotTokenDBErr(t *testing.T) {
	t.Parallel()
	ctx, cancel := testContext(t)
	defer cancel()

	tw := &twitchfakes.FakeAPI{}
	db := pool.FreshDB(t)
	defer db.Close()

	breakDB(t, db)

	_, err := twitchx.FindBotToken(ctx, db, tw, "hortbot")
	assert.ErrorContains(t, err, "does not exist")
}

func TestFindBotTokenNew(t *testing.T) {
	t.Parallel()
	ctx, cancel := testContext(t)
	defer cancel()

	newT := &models.TwitchToken{
		TwitchID:     1234,
		BotName:      null.StringFrom("hortbot"),
		AccessToken:  "at-asdfghjkl0987654321",
		TokenType:    "Bearer",
		RefreshToken: "rt-asdfghjkl0987654321",
		Expiry:       time.Now(),
	}

	newTok := modelsx.ModelToToken(newT)

	tw := &twitchfakes.FakeAPI{
		GetUserForTokenStub: func(context.Context, *oauth2.Token) (*twitch.User, *oauth2.Token, error) {
			return &twitch.User{
				ID: twitch.IDStr(newT.ID),
			}, newTok, nil
		},
	}
	db := pool.FreshDB(t)
	defer db.Close()

	orig := &models.TwitchToken{
		TwitchID:     1234,
		BotName:      null.StringFrom("hortbot"),
		AccessToken:  "at-qwertyuiop1234567890",
		TokenType:    "Bearer",
		RefreshToken: "rt-qwertyuiop1234567890",
		Expiry:       time.Now(),
	}

	assert.NilError(t, orig.Insert(ctx, db, boil.Infer()))

	tok, err := twitchx.FindBotToken(ctx, db, tw, "hortbot")
	assert.NilError(t, err)
	assert.Assert(t, cmp.DeepEqual(newTok, tok, tokenCmp))

	upserted, err := models.TwitchTokens(models.TwitchTokenWhere.TwitchID.EQ(1234)).One(ctx, db)
	assert.NilError(t, err)

	upsertedTok := modelsx.ModelToToken(upserted)
	assert.Assert(t, cmp.DeepEqual(newTok, upsertedTok, tokenCmp))
}

func TestFindBotTokenNewDBError(t *testing.T) {
	t.Parallel()
	ctx, cancel := testContext(t)
	defer cancel()

	newT := &models.TwitchToken{
		TwitchID:     1234,
		BotName:      null.StringFrom("hortbot"),
		AccessToken:  "at-asdfghjkl0987654321",
		TokenType:    "Bearer",
		RefreshToken: "rt-asdfghjkl0987654321",
		Expiry:       time.Now(),
	}

	newTok := modelsx.ModelToToken(newT)

	db := pool.FreshDB(t)
	defer db.Close()
	tw := &twitchfakes.FakeAPI{
		GetUserForTokenStub: func(context.Context, *oauth2.Token) (*twitch.User, *oauth2.Token, error) {
			breakDB(t, db)

			return &twitch.User{
				ID: twitch.IDStr(newT.ID),
			}, newTok, nil
		},
	}

	orig := &models.TwitchToken{
		TwitchID:     1234,
		BotName:      null.StringFrom("hortbot"),
		AccessToken:  "at-qwertyuiop1234567890",
		TokenType:    "Bearer",
		RefreshToken: "rt-qwertyuiop1234567890",
		Expiry:       time.Now(),
	}

	assert.NilError(t, orig.Insert(ctx, db, boil.Infer()))

	_, err := twitchx.FindBotToken(ctx, db, tw, "hortbot")
	assert.ErrorContains(t, err, "does not exist")
}

func breakDB(t testing.TB, db *sql.DB) {
	_, err := db.Exec("DROP TABLE twitch_tokens")
	assert.NilError(t, err)
}

func testContext(t testing.TB) (context.Context, context.CancelFunc) {
	t.Helper()
	ctx := ctxlog.WithLogger(context.Background(), testutil.Logger(t))
	return context.WithTimeout(ctx, 5*time.Minute)
}
