// Package twitchx provides extensions to the Twitch client.
package twitchx

import (
	"context"
	"database/sql"

	"github.com/hortbot/hortbot/internal/db/models"
	"github.com/hortbot/hortbot/internal/db/modelsx"
	"github.com/hortbot/hortbot/internal/pkg/apis/twitch"
	"github.com/volatiletech/null"
	"github.com/volatiletech/sqlboiler/boil"
	"golang.org/x/oauth2"
)

// FindBotToken looks for the bot's token in the database, and fetches a token
// if one has not yet been found.
func FindBotToken(ctx context.Context, db boil.ContextExecutor, tw twitch.API, botName string) (*oauth2.Token, error) {
	_, tok, err := findBotToken(ctx, db, tw, botName)
	return tok, err
}

func findBotToken(ctx context.Context, db boil.ContextExecutor, tw twitch.API, botName string) (old, new *oauth2.Token, err error) {
	botNameNull := null.StringFrom(botName)

	token, err := models.TwitchTokens(models.TwitchTokenWhere.BotName.EQ(botNameNull)).One(ctx, db)
	switch err {
	case nil:
	case sql.ErrNoRows:
		return nil, nil, nil
	default:
		return nil, nil, err
	}

	tok := modelsx.ModelToToken(token)
	user, newTok, err := tw.GetUserForToken(ctx, tok)
	if err != nil {
		return nil, nil, err
	}

	if newTok == nil {
		return tok, tok, nil
	}

	if err := modelsx.UpsertToken(ctx, db, modelsx.TokenToModel(user.ID, tok)); err != nil {
		return nil, nil, err
	}

	return tok, newTok, nil
}
