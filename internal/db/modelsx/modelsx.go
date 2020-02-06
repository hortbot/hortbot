// Package modelsx provides extensions for the models package.
package modelsx

import (
	"context"
	"database/sql"

	"github.com/hortbot/hortbot/internal/db/models"
	"github.com/volatiletech/null"
	"github.com/volatiletech/sqlboiler/boil"
	"github.com/volatiletech/sqlboiler/queries"
	"github.com/volatiletech/sqlboiler/queries/qm"
	"golang.org/x/oauth2"
)

// TokenToModel converts a Twitch user's oauth2 token to a model for insertion.
func TokenToModel(id int64, tok *oauth2.Token) *models.TwitchToken {
	return &models.TwitchToken{
		TwitchID:     id,
		AccessToken:  tok.AccessToken,
		TokenType:    tok.TokenType,
		RefreshToken: tok.RefreshToken,
		Expiry:       tok.Expiry,
	}
}

// ModelToToken converts a token model to an oauth2 token for use in an HTTP client.
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

// UpsertToken inserts a token into the database, or updates it if the token has been changed.
func UpsertToken(ctx context.Context, exec boil.ContextExecutor, tt *models.TwitchToken) error {
	return tt.Upsert(ctx, exec, true, []string{models.TwitchTokenColumns.TwitchID}, tokenUpdate, boil.Infer())
}

var fullTokenUpdate = boil.Whitelist(
	models.TwitchTokenColumns.UpdatedAt,
	models.TwitchTokenColumns.BotName,
	models.TwitchTokenColumns.AccessToken,
	models.TwitchTokenColumns.TokenType,
	models.TwitchTokenColumns.RefreshToken,
	models.TwitchTokenColumns.Expiry,
)

// FullUpsertToken inserts the token into the database, or inserts all columns as written in the model.
func FullUpsertToken(ctx context.Context, exec boil.ContextExecutor, tt *models.TwitchToken) error {
	return tt.Upsert(ctx, exec, true, []string{models.TwitchTokenColumns.TwitchID}, fullTokenUpdate, boil.Infer())
}

// DeleteCommandInfo deletes all references to a specific command from the database.
func DeleteCommandInfo(ctx context.Context, exec boil.ContextExecutor, info *models.CommandInfo) (repeated *models.RepeatedCommand, scheduled *models.ScheduledCommand, err error) {
	repeated, err = info.RepeatedCommand().One(ctx, exec)
	if err != nil {
		if err != sql.ErrNoRows {
			return nil, nil, err
		}
	} else {
		if err := repeated.Delete(ctx, exec); err != nil {
			return nil, nil, err
		}
	}

	scheduled, err = info.ScheduledCommand().One(ctx, exec)
	if err != nil {
		if err != sql.ErrNoRows {
			return nil, nil, err
		}
	} else {
		if err := scheduled.Delete(ctx, exec); err != nil {
			return nil, nil, err
		}
	}

	if err := info.Delete(ctx, exec); err != nil {
		return nil, nil, err
	}

	if command := info.R.CustomCommand; command != nil {
		if err := command.Delete(ctx, exec); err != nil {
			return nil, nil, err
		}
	}

	if list := info.R.CommandList; list != nil {
		if err := list.Delete(ctx, exec); err != nil {
			return nil, nil, err
		}
	}

	return repeated, scheduled, nil
}

const findCommandQueryUpdate = `
SELECT command_infos.*, custom_commands.message
FROM command_infos
LEFT OUTER JOIN custom_commands on custom_commands.id = command_infos.custom_command_id
WHERE ("command_infos"."channel_id" = $1) AND ("command_infos"."name" = $2)
FOR UPDATE OF command_infos
`

const findCommandQuery = `
SELECT command_infos.*, custom_commands.message
FROM command_infos
LEFT OUTER JOIN custom_commands on custom_commands.id = command_infos.custom_command_id
WHERE ("command_infos"."channel_id" = $1) AND ("command_infos"."name" = $2)
`

// FindCommand finds a command for a given channel.
func FindCommand(ctx context.Context, exec boil.Executor, id int64, name string, forUpdate bool) (info *models.CommandInfo, commandMsg null.String, found bool, err error) {
	infoAndCommand := struct {
		CommandInfo models.CommandInfo `boil:"command_infos,bind"`
		Message     null.String        `boil:"message"`
	}{}

	query := findCommandQuery
	if forUpdate {
		query = findCommandQueryUpdate
	}

	// This is much faster than using qm.Load, as SQLBoiler's loading does multiple
	// queries to fetch 1:1 relationships rather than joins.
	err = queries.Raw(query, id, name).Bind(ctx, exec, &infoAndCommand)

	if err == sql.ErrNoRows {
		return nil, null.String{}, false, nil
	}

	if err != nil {
		return nil, null.String{}, false, err
	}

	return &infoAndCommand.CommandInfo, infoAndCommand.Message, true, nil
}

// ListActiveChannels returns a list of active IRC channels (with # prefix) for the specified bot.
func ListActiveChannels(ctx context.Context, exec boil.Executor, botName string) ([]string, error) {
	var channels []struct {
		Name string
	}

	err := models.Channels(
		qm.Select(models.ChannelColumns.Name),
		models.ChannelWhere.Active.EQ(true),
		models.ChannelWhere.BotName.EQ(botName),
	).Bind(ctx, exec, &channels)
	if err != nil {
		return nil, err
	}

	out := make([]string, len(channels), len(channels)+1)

	for i, c := range channels {
		out[i] = "#" + c.Name
	}

	out = append(out, "#"+botName)

	return out, nil
}

// DeleteChannel deletes a channel and every row in every table which
// references it. Must be done in a transaction, as the steps it takes are
// extremely likely to voilate constraints.
//
// Ensure other state (like repeats) are refreshed after using this function
// (or otherwise do not take effect if their data is not found).
//
// Must be kept in sync with migrations to keep this effective.
func DeleteChannel(ctx context.Context, exec boil.ContextExecutor, id int64) error {
	// Reverse table order; shouldn't really matter when run in a transaction.
	queries := []interface {
		DeleteAll(context.Context, boil.ContextExecutor) error
	}{
		models.ScheduledCommands(models.ScheduledCommandWhere.ChannelID.EQ(id)),
		models.RepeatedCommands(models.RepeatedCommandWhere.ChannelID.EQ(id)),
		models.CommandInfos(models.CommandInfoWhere.ChannelID.EQ(id)),
		models.CommandLists(models.CommandListWhere.ChannelID.EQ(id)),
		models.Variables(models.VariableWhere.ChannelID.EQ(id)),
		models.Autoreplies(models.AutoreplyWhere.ChannelID.EQ(id)),
		models.Quotes(models.QuoteWhere.ChannelID.EQ(id)),
		models.CustomCommands(models.CustomCommandWhere.ChannelID.EQ(id)),
		models.Channels(models.ChannelWhere.ID.EQ(id)),
	}

	for _, q := range queries {
		if err := q.DeleteAll(ctx, exec); err != nil {
			return err
		}
	}

	return nil
}
