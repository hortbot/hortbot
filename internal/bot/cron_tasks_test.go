package bot

import (
	"context"
	"testing"
	"time"

	"github.com/hortbot/hortbot/internal/db/dbsql"
	"github.com/hortbot/hortbot/internal/db/migrations"
	"github.com/hortbot/hortbot/internal/pkg/apiclient/twitch"
	"github.com/hortbot/hortbot/internal/pkg/apiclient/twitch/twitchmocks"
	"github.com/hortbot/hortbot/internal/pkg/testpostgres"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/oauth2"
	"gotest.tools/v3/assert"
)

func TestValidateTokensUsesTransactionConnection(t *testing.T) {
	t.Parallel()

	pdb, err := testpostgres.New()
	assert.NilError(t, err)
	t.Cleanup(pdb.Cleanup)
	assert.NilError(t, migrations.Up(pdb.ConnStr(), nil))

	config, err := pgxpool.ParseConfig(pdb.ConnStr())
	assert.NilError(t, err)
	config.MaxConns = 1
	db, err := pgxpool.NewWithConfig(t.Context(), config)
	assert.NilError(t, err)
	t.Cleanup(db.Close)

	queries := dbsql.New(db)
	_, err = queries.UpsertTwitchToken(t.Context(), dbsql.UpsertTwitchTokenParams{
		TwitchID:     1,
		BotName:      pgtype.Text{},
		AccessToken:  "access",
		TokenType:    "bearer",
		RefreshToken: "refresh",
		Expiry:       dbsql.TimestamptzFrom(time.Now().Add(time.Hour)),
		Scopes:       []string{"old"},
	})
	assert.NilError(t, err)

	twitchAPI := &twitchmocks.APIMock{
		ValidateFunc: func(context.Context, *oauth2.Token) (*twitch.Validation, *oauth2.Token, error) {
			return &twitch.Validation{Scopes: []string{"new"}}, nil, nil
		},
	}
	b := &Bot{
		db:      db,
		queries: queries,
		deps:    &sharedDeps{Twitch: twitchAPI},
	}

	ctx, cancel := context.WithTimeout(t.Context(), time.Second)
	defer cancel()
	assert.NilError(t, b.validateTokens(ctx, false))

	token, err := queries.GetTwitchTokenByID(t.Context(), 1)
	assert.NilError(t, err)
	assert.DeepEqual(t, token.Scopes, []string{"new"})
}
