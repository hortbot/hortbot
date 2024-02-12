// Package twitchx provides extensions to the Twitch client.
package twitchx

import (
	"context"
	"database/sql"

	"github.com/hortbot/hortbot/internal/db/models"
	"github.com/hortbot/hortbot/internal/db/modelsx"
	"github.com/hortbot/hortbot/internal/pkg/apiclient/twitch"
	"github.com/volatiletech/null/v8"
	"github.com/volatiletech/sqlboiler/v4/boil"
	"golang.org/x/oauth2"
)

// FindBotToken looks for the bot's token in the database, and fetches a token
// if one has not yet been found.
func FindBotToken(ctx context.Context, db boil.ContextExecutor, tw twitch.API, botName string) (*oauth2.Token, error) {
	botNameNull := null.StringFrom(botName)

	token, err := models.TwitchTokens(models.TwitchTokenWhere.BotName.EQ(botNameNull)).One(ctx, db)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil //nolint:nilnil
		}
		return nil, err
	}

	// Validate the token before trying to use it; this is only called for IRC,
	// which will just fall over when given incorrect credentials.
	tok := modelsx.ModelToToken(token)
	validation, newTok, err := tw.Validate(ctx, tok)
	if err != nil {
		return nil, err
	}
	if newTok == nil {
		return tok, nil
	}

	newToken := modelsx.TokenToModel(newTok, token.TwitchID, botNameNull, validation.Scopes)
	if err := modelsx.UpsertToken(ctx, db, newToken); err != nil {
		return nil, err
	}

	return newTok, nil
}
