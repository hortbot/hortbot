// Package modelsx provides extensions for the models package.
package modelsx

import (
	"context"
	"database/sql"
	"errors"

	"github.com/hortbot/hortbot/internal/db/models"
	"github.com/volatiletech/null/v8"
	"github.com/volatiletech/sqlboiler/v4/boil"
	"github.com/volatiletech/sqlboiler/v4/queries"
	"golang.org/x/oauth2"
)

// TokenToModel converts a Twitch user's oauth2 token to a model for insertion.
func TokenToModel(tok *oauth2.Token, id int64, botName null.String, scopes []string) *models.TwitchToken {
	return &models.TwitchToken{
		TwitchID:     id,
		BotName:      botName,
		AccessToken:  tok.AccessToken,
		TokenType:    tok.TokenType,
		RefreshToken: tok.RefreshToken,
		Expiry:       tok.Expiry,
		Scopes:       scopes,
	}
}

// TokenToModelWithoutPreservedColumns converts a Twitch user's oauth2 token to a model for insertion,
// without BotName and Scopes.
func TokenToModelWithoutPreservedColumns(tok *oauth2.Token, id int64) *models.TwitchToken {
	return TokenToModel(tok, id, null.String{}, nil)
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

var tokenConflictColumns = []string{models.TwitchTokenColumns.TwitchID}

var withoutCreatedAt = boil.Blacklist(models.TwitchTokenColumns.CreatedAt)

// UpsertToken inserts the token into the database, or inserts all columns as written in the model.
func UpsertToken(ctx context.Context, exec boil.ContextExecutor, tt *models.TwitchToken) error {
	return tt.Upsert(ctx, exec, true, tokenConflictColumns, withoutCreatedAt, boil.Infer())
}

var withoutPreservedColumns = boil.Blacklist(
	models.TwitchTokenColumns.CreatedAt,
	models.TwitchTokenColumns.BotName,
	models.TwitchTokenColumns.Scopes,
)

// UpsertTokenWithoutPreservedColumns upserts a token without BotName and Scopes.
func UpsertTokenWithoutPreservedColumns(ctx context.Context, exec boil.ContextExecutor, tt *models.TwitchToken) error {
	return tt.Upsert(ctx, exec, true, tokenConflictColumns, withoutPreservedColumns, boil.Infer())
}

// DeleteCommandInfo deletes all references to a specific command from the database.
func DeleteCommandInfo(ctx context.Context, exec boil.ContextExecutor, info *models.CommandInfo) (repeated *models.RepeatedCommand, scheduled *models.ScheduledCommand, err error) {
	repeated, err = info.RepeatedCommand().One(ctx, exec)
	if err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			return nil, nil, err
		}
	} else {
		if err := repeated.Delete(ctx, exec); err != nil {
			return nil, nil, err
		}
	}

	scheduled, err = info.ScheduledCommand().One(ctx, exec)
	if err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
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

	if errors.Is(err, sql.ErrNoRows) {
		return nil, null.String{}, false, nil
	}

	if err != nil {
		return nil, null.String{}, false, err
	}

	return &infoAndCommand.CommandInfo, infoAndCommand.Message, true, nil
}

func GetBots(ctx context.Context, exec boil.ContextExecutor) (map[string]int64, map[int64]string, error) {
	bots, err := models.TwitchTokens(models.TwitchTokenWhere.BotName.IsNotNull()).All(ctx, exec)
	if err != nil {
		return nil, nil, err
	}

	botNameToID := make(map[string]int64, len(bots))
	botIDToName := make(map[int64]string, len(bots))
	for _, bot := range bots {
		botNameToID[bot.BotName.String] = bot.TwitchID
		botIDToName[bot.TwitchID] = bot.BotName.String
	}

	return botNameToID, botIDToName, nil
}

type activeChannels struct {
	// IRC maps bot names to IRC channel names.
	IRC map[string][]string

	// EventSub maps bot user IDs to broadcaster IDs.
	EventSub map[int64][]int64
}

const listActiveChannelsQuery = `
SELECT c.twitch_id, c.name, c.bot_name, 'channel:bot' = ANY(tt.scopes) as has_auth, m.id IS NOT NULL as has_mod
FROM channels c
LEFT OUTER JOIN twitch_tokens tt ON tt.twitch_id = c.twitch_id
LEFT OUTER JOIN moderated_channels m ON m.broadcaster_id = c.twitch_id AND m.bot_name = c.bot_name
WHERE c.active
`

const useEventSub = true

func listActiveChannels(ctx context.Context, exec boil.ContextExecutor) (*activeChannels, error) {
	botNameToID, _, err := GetBots(ctx, exec)
	if err != nil {
		return nil, err
	}

	var rows []*struct {
		TwitchID int64     `boil:"twitch_id"`
		Name     string    `boil:"name"`
		BotName  string    `boil:"bot_name"`
		HasAuth  null.Bool `boil:"has_auth"`
		HasMod   null.Bool `boil:"has_mod"`
	}

	if err := queries.Raw(listActiveChannelsQuery).Bind(ctx, exec, &rows); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return &activeChannels{}, nil
		}
		return nil, err
	}

	irc := make(map[string][]string, len(rows))
	eventSub := make(map[int64][]int64, len(rows))

	for botName, botID := range botNameToID {
		irc[botName] = []string{"#" + botName}
		eventSub[botID] = []int64{botID}
	}

	for _, row := range rows {
		botName := row.BotName
		if row.HasAuth.Bool || row.HasMod.Bool {
			if useEventSub {
				botID := botNameToID[botName]
				eventSub[botID] = append(eventSub[botID], row.TwitchID)
				continue
			}
			irc[botName] = append(irc[botName], "#"+row.Name)
		}
	}

	return &activeChannels{
		IRC:      irc,
		EventSub: eventSub,
	}, nil
}

func ListActiveIRCChannels(ctx context.Context, exec boil.ContextExecutor, botName string) ([]string, error) {
	active, err := listActiveChannels(ctx, exec)
	if err != nil {
		return nil, err
	}
	return active.IRC[botName], nil
}

func ListActiveEventSubChannels(ctx context.Context, exec boil.ContextExecutor) (map[int64][]int64, error) {
	active, err := listActiveChannels(ctx, exec)
	if err != nil {
		return nil, err
	}
	return active.EventSub, nil
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
