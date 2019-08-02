package modelsx

import (
	"context"

	"github.com/hortbot/hortbot/internal/db/models"
	"github.com/volatiletech/sqlboiler/boil"
	"golang.org/x/oauth2"
)

func TokenToModel(id int64, tok *oauth2.Token) *models.TwitchToken {
	return &models.TwitchToken{
		TwitchID:     id,
		AccessToken:  tok.AccessToken,
		TokenType:    tok.TokenType,
		RefreshToken: tok.RefreshToken,
		Expiry:       tok.Expiry,
	}
}

func ModelToToken(tt *models.TwitchToken) *oauth2.Token {
	return &oauth2.Token{
		AccessToken:  tt.AccessToken,
		TokenType:    tt.TokenType,
		RefreshToken: tt.RefreshToken,
		Expiry:       tt.Expiry,
	}
}

var tokenUpdate = boil.Whitelist(
	models.TwitchTokenColumns.UpdatedAt,
	models.TwitchTokenColumns.AccessToken,
	models.TwitchTokenColumns.TokenType,
	models.TwitchTokenColumns.RefreshToken,
	models.TwitchTokenColumns.Expiry,
)

func UpsertToken(ctx context.Context, exec boil.ContextExecutor, tt *models.TwitchToken) error {
	return tt.Upsert(ctx, exec, true, []string{models.TwitchTokenColumns.TwitchID}, tokenUpdate, boil.Infer())
}
