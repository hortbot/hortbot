package modelsx

import (
	"context"
	"database/sql"

	"github.com/hortbot/hortbot/internal/db/models"
	"github.com/volatiletech/null"
	"github.com/volatiletech/sqlboiler/boil"
	"github.com/volatiletech/sqlboiler/queries"
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
	models.TwitchTokenColumns.BotName,
	models.TwitchTokenColumns.AccessToken,
	models.TwitchTokenColumns.TokenType,
	models.TwitchTokenColumns.RefreshToken,
	models.TwitchTokenColumns.Expiry,
)

func UpsertToken(ctx context.Context, exec boil.ContextExecutor, tt *models.TwitchToken) error {
	return tt.Upsert(ctx, exec, true, []string{models.TwitchTokenColumns.TwitchID}, tokenUpdate, boil.Infer())
}

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
