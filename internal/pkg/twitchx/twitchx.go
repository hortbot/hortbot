// Package twitchx provides extensions to the Twitch client.
package twitchx

import (
	"context"
	"database/sql"

	"github.com/hortbot/hortbot/internal/db/models"
	"github.com/hortbot/hortbot/internal/db/modelsx"
	"github.com/hortbot/hortbot/internal/pkg/apiclient/twitch"
	"github.com/volatiletech/null"
	"github.com/volatiletech/sqlboiler/boil"
	"golang.org/x/oauth2"
)

// FindBotToken looks for the bot's token in the database, and fetches a token
// if one has not yet been found.
func FindBotToken(ctx context.Context, db boil.ContextExecutor, tw twitch.API, botName string) (*oauth2.Token, error) {
	botNameNull := null.StringFrom(botName)

	token, err := models.TwitchTokens(models.TwitchTokenWhere.BotName.EQ(botNameNull)).One(ctx, db)
	switch err {
	case nil:
	case sql.ErrNoRows:
		return nil, nil
	default:
		return nil, err
	}

	tok := modelsx.ModelToToken(token)
	_, newTok, err := tw.GetUserForToken(ctx, tok)
	if err != nil {
		return nil, err
	}

	if newTok == nil {
		return tok, nil
	}

	token.AccessToken = newTok.AccessToken
	token.TokenType = newTok.TokenType
	token.RefreshToken = newTok.RefreshToken
	token.Expiry = newTok.Expiry

	if err := modelsx.FullUpsertToken(ctx, db, token); err != nil {
		return nil, err
	}

	return newTok, nil
}
