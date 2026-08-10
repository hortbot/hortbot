package dbsql

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"golang.org/x/oauth2"
)

func TextFrom(value string) pgtype.Text {
	return pgtype.Text{String: value, Valid: true}
}

func Int4From(value int32) pgtype.Int4 {
	return pgtype.Int4{Int32: value, Valid: true}
}

func Int8From(value int64) pgtype.Int8 {
	return pgtype.Int8{Int64: value, Valid: true}
}

func TimestamptzFrom(value time.Time) pgtype.Timestamptz {
	return pgtype.Timestamptz{Time: value, Valid: true}
}

func (q *Queries) SaveChannelMembership(ctx context.Context, channel *Channel) error {
	return q.UpdateChannelMembership(ctx, UpdateChannelMembershipParams{
		Active:      channel.Active,
		BotName:     channel.BotName,
		Name:        channel.Name,
		DisplayName: channel.DisplayName,
		ID:          channel.ID,
	})
}

func (q *Queries) SaveChannelSettings(ctx context.Context, channel *Channel) error {
	return q.UpdateChannelSettings(ctx, UpdateChannelSettingsParams{
		Prefix:                      channel.Prefix,
		Bullet:                      channel.Bullet,
		Mode:                        channel.Mode,
		Cooldown:                    channel.Cooldown,
		LastFM:                      channel.LastFM,
		ParseYoutube:                channel.ParseYoutube,
		ExtraLifeID:                 channel.ExtraLifeID,
		SteamID:                     channel.SteamID,
		UrbanEnabled:                channel.UrbanEnabled,
		Tweet:                       channel.Tweet,
		RollLevel:                   channel.RollLevel,
		RollCooldown:                channel.RollCooldown,
		RollDefault:                 channel.RollDefault,
		ShouldModerate:              channel.ShouldModerate,
		DisplayWarnings:             channel.DisplayWarnings,
		EnableWarnings:              channel.EnableWarnings,
		TimeoutDuration:             channel.TimeoutDuration,
		EnableFilters:               channel.EnableFilters,
		FilterLinks:                 channel.FilterLinks,
		PermittedLinks:              channel.PermittedLinks,
		SubsMayLink:                 channel.SubsMayLink,
		FilterCaps:                  channel.FilterCaps,
		FilterCapsMinChars:          channel.FilterCapsMinChars,
		FilterCapsPercentage:        channel.FilterCapsPercentage,
		FilterCapsMinCaps:           channel.FilterCapsMinCaps,
		FilterEmotes:                channel.FilterEmotes,
		FilterEmotesMax:             channel.FilterEmotesMax,
		FilterEmotesSingle:          channel.FilterEmotesSingle,
		FilterSymbols:               channel.FilterSymbols,
		FilterSymbolsPercentage:     channel.FilterSymbolsPercentage,
		FilterSymbolsMinSymbols:     channel.FilterSymbolsMinSymbols,
		FilterMe:                    channel.FilterMe,
		FilterMaxLength:             channel.FilterMaxLength,
		FilterBannedPhrases:         channel.FilterBannedPhrases,
		FilterBannedPhrasesPatterns: channel.FilterBannedPhrasesPatterns,
		FilterExemptLevel:           channel.FilterExemptLevel,
		ID:                          channel.ID,
	})
}

func NewTwitchToken(token *oauth2.Token, twitchID int64, botName pgtype.Text, scopes []string) *TwitchToken {
	return &TwitchToken{
		TwitchID:     twitchID,
		BotName:      botName,
		AccessToken:  token.AccessToken,
		TokenType:    token.TokenType,
		RefreshToken: token.RefreshToken,
		Expiry:       TimestamptzFrom(token.Expiry),
		Scopes:       scopes,
	}
}

func (t TwitchToken) OAuth2Token() *oauth2.Token {
	return &oauth2.Token{
		AccessToken:  t.AccessToken,
		TokenType:    t.TokenType,
		RefreshToken: t.RefreshToken,
		Expiry:       t.Expiry.Time,
	}
}

func (q *Queries) SaveTwitchToken(ctx context.Context, token *TwitchToken) error {
	updated, err := q.UpsertTwitchToken(ctx, UpsertTwitchTokenParams{
		TwitchID:     token.TwitchID,
		BotName:      token.BotName,
		AccessToken:  token.AccessToken,
		TokenType:    token.TokenType,
		RefreshToken: token.RefreshToken,
		Expiry:       token.Expiry,
		Scopes:       token.Scopes,
	})
	if err != nil {
		return fmt.Errorf("upserting Twitch token: %w", err)
	}
	*token = updated
	return nil
}

func (q *Queries) SaveTwitchTokenPreservingMetadata(ctx context.Context, token *TwitchToken) error {
	updated, err := q.UpsertTwitchTokenPreservingMetadata(ctx, UpsertTwitchTokenPreservingMetadataParams{
		TwitchID:     token.TwitchID,
		AccessToken:  token.AccessToken,
		TokenType:    token.TokenType,
		RefreshToken: token.RefreshToken,
		Expiry:       token.Expiry,
	})
	if err != nil {
		return fmt.Errorf("upserting Twitch token: %w", err)
	}
	*token = updated
	return nil
}

func (q *Queries) DeleteCommandInfoCascade(ctx context.Context, info *CommandInfo) (repeated *RepeatedCommand, scheduled *ScheduledCommand, err error) {
	if id, queryErr := q.GetRepeatedCommandIDByInfo(ctx, info.ID); queryErr == nil {
		repeated = &RepeatedCommand{ID: id}
	} else if !errors.Is(queryErr, pgx.ErrNoRows) {
		return nil, nil, fmt.Errorf("getting repeated command: %w", queryErr)
	}
	if id, queryErr := q.GetScheduledCommandIDByInfo(ctx, info.ID); queryErr == nil {
		scheduled = &ScheduledCommand{ID: id}
	} else if !errors.Is(queryErr, pgx.ErrNoRows) {
		return nil, nil, fmt.Errorf("getting scheduled command: %w", queryErr)
	}
	if err := q.DeleteRepeatedCommandByInfo(ctx, info.ID); err != nil {
		return nil, nil, fmt.Errorf("deleting repeated command: %w", err)
	}
	if err := q.DeleteScheduledCommandByInfo(ctx, info.ID); err != nil {
		return nil, nil, fmt.Errorf("deleting scheduled command: %w", err)
	}
	if err := q.DeleteCommandInfo(ctx, info.ID); err != nil {
		return nil, nil, fmt.Errorf("deleting command info: %w", err)
	}
	if info.CustomCommandID.Valid {
		if err := q.DeleteCustomCommand(ctx, info.CustomCommandID.Int64); err != nil {
			return nil, nil, fmt.Errorf("deleting custom command: %w", err)
		}
	}
	if info.CommandListID.Valid {
		if err := q.DeleteCommandList(ctx, info.CommandListID.Int64); err != nil {
			return nil, nil, fmt.Errorf("deleting command list: %w", err)
		}
	}
	return repeated, scheduled, nil
}

func (q *Queries) LookupCommand(ctx context.Context, channelID int64, name string, forUpdate bool) (*CommandInfo, pgtype.Text, bool, error) {
	params := FindCommandParams{ChannelID: channelID, Name: name}
	var row FindCommandRow
	var err error
	if forUpdate {
		locked, queryErr := q.FindCommandForUpdate(ctx, FindCommandForUpdateParams(params))
		if queryErr != nil {
			err = queryErr
		} else {
			row = FindCommandRow(locked)
		}
	} else {
		row, err = q.FindCommand(ctx, params)
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, pgtype.Text{}, false, nil
	}
	if err != nil {
		return nil, pgtype.Text{}, false, fmt.Errorf("finding command: %w", err)
	}
	return &CommandInfo{
		ID: row.ID, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt,
		ChannelID: row.ChannelID, Name: row.Name, AccessLevel: row.AccessLevel,
		Count: row.Count, Creator: row.Creator, Editor: row.Editor,
		LastUsed: row.LastUsed, CustomCommandID: row.CustomCommandID,
		CommandListID: row.CommandListID,
	}, row.Message, true, nil
}

func (q *Queries) BotMaps(ctx context.Context) (map[string]int64, map[int64]string, error) {
	bots, err := q.ListBotTwitchTokens(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("getting bot tokens: %w", err)
	}
	botNameToID := make(map[string]int64, len(bots))
	botIDToName := make(map[int64]string, len(bots))
	for _, bot := range bots {
		botNameToID[bot.BotName.String] = bot.TwitchID
		botIDToName[bot.TwitchID] = bot.BotName.String
	}
	return botNameToID, botIDToName, nil
}

func (q *Queries) ActiveChannelsByBot(ctx context.Context) (map[int64][]int64, error) {
	botNameToID, _, err := q.BotMaps(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := q.ListActiveChannelAssignments(ctx)
	if err != nil {
		return nil, fmt.Errorf("listing active channels: %w", err)
	}
	botToChannels := make(map[int64][]int64, len(rows))
	for _, botID := range botNameToID {
		botToChannels[botID] = []int64{botID}
	}
	for _, row := range rows {
		botID := botNameToID[row.BotName]
		botToChannels[botID] = append(botToChannels[botID], row.TwitchID)
	}
	return botToChannels, nil
}

func (q *Queries) DeleteChannelCascade(ctx context.Context, id int64) error {
	deletes := []func(context.Context, int64) error{
		q.DeleteScheduledCommandsByChannel,
		q.DeleteRepeatedCommandsByChannel,
		q.DeleteCommandInfosByChannel,
		q.DeleteCommandListsByChannel,
		q.DeleteVariablesByChannel,
		q.DeleteAutorepliesByChannel,
		q.DeleteQuotesByChannel,
		q.DeleteCustomCommandsByChannel,
		q.DeleteHighlightsByChannel,
		q.DeleteChannel,
	}
	for _, deleteRows := range deletes {
		if err := deleteRows(ctx, id); err != nil {
			return fmt.Errorf("deleting channel data: %w", err)
		}
	}
	return nil
}
