package graph

// This file will be automatically regenerated based on the schema, any resolver implementations
// will be copied through when generating and any unknown code will be moved to the end.

import (
	"context"
	"database/sql"
	"strconv"
	"time"

	"github.com/hortbot/hortbot/internal/db/graph/generated"
	"github.com/hortbot/hortbot/internal/db/models"
)

func (r *autoreplyResolver) Channel(ctx context.Context, obj *models.Autoreply) (*models.Channel, error) {
	tx := txFromContext(ctx)
	return obj.Channel().One(ctx, tx)
}

func (r *autoreplyResolver) OrigPattern(ctx context.Context, obj *models.Autoreply) (*string, error) {
	return obj.OrigPattern.Ptr(), nil
}

func (r *channelResolver) Bullet(ctx context.Context, obj *models.Channel) (*string, error) {
	return obj.Bullet.Ptr(), nil
}

func (r *channelResolver) MessageCount(ctx context.Context, obj *models.Channel) (string, error) {
	return strconv.FormatInt(obj.MessageCount, 10), nil
}

func (r *channelResolver) Ignored(ctx context.Context, obj *models.Channel) ([]string, error) {
	return obj.Ignored, nil
}

func (r *channelResolver) CustomOwners(ctx context.Context, obj *models.Channel) ([]string, error) {
	return obj.CustomOwners, nil
}

func (r *channelResolver) CustomMods(ctx context.Context, obj *models.Channel) ([]string, error) {
	return obj.CustomMods, nil
}

func (r *channelResolver) CustomRegulars(ctx context.Context, obj *models.Channel) ([]string, error) {
	return obj.CustomRegulars, nil
}

func (r *channelResolver) Cooldown(ctx context.Context, obj *models.Channel) (*int, error) {
	return obj.Cooldown.Ptr(), nil
}

func (r *channelResolver) PermittedLinks(ctx context.Context, obj *models.Channel) ([]string, error) {
	return obj.PermittedLinks, nil
}

func (r *channelResolver) FilterBannedPhrasesPatterns(ctx context.Context, obj *models.Channel) ([]string, error) {
	return obj.FilterBannedPhrasesPatterns, nil
}

func (r *channelResolver) Autoreplies(ctx context.Context, obj *models.Channel) ([]*models.Autoreply, error) {
	tx := txFromContext(ctx)
	return obj.Autoreplies().All(ctx, tx)
}

func (r *channelResolver) CommandInfos(ctx context.Context, obj *models.Channel) ([]*models.CommandInfo, error) {
	tx := txFromContext(ctx)
	return obj.CommandInfos().All(ctx, tx)
}

func (r *channelResolver) CommandLists(ctx context.Context, obj *models.Channel) ([]*models.CommandList, error) {
	tx := txFromContext(ctx)
	return obj.CommandLists().All(ctx, tx)
}

func (r *channelResolver) CustomCommands(ctx context.Context, obj *models.Channel) ([]*models.CustomCommand, error) {
	tx := txFromContext(ctx)
	return obj.CustomCommands().All(ctx, tx)
}

func (r *channelResolver) Highlights(ctx context.Context, obj *models.Channel) ([]*models.Highlight, error) {
	tx := txFromContext(ctx)
	return obj.Highlights().All(ctx, tx)
}

func (r *channelResolver) Quotes(ctx context.Context, obj *models.Channel) ([]*models.Quote, error) {
	tx := txFromContext(ctx)
	return obj.Quotes().All(ctx, tx)
}

func (r *channelResolver) RepeatedCommands(ctx context.Context, obj *models.Channel) ([]*models.RepeatedCommand, error) {
	tx := txFromContext(ctx)
	return obj.RepeatedCommands().All(ctx, tx)
}

func (r *channelResolver) ScheduledCommands(ctx context.Context, obj *models.Channel) ([]*models.ScheduledCommand, error) {
	tx := txFromContext(ctx)
	return obj.ScheduledCommands().All(ctx, tx)
}

func (r *channelResolver) Variables(ctx context.Context, obj *models.Channel) ([]*models.Variable, error) {
	tx := txFromContext(ctx)
	return obj.Variables().All(ctx, tx)
}

func (r *commandInfoResolver) Count(ctx context.Context, obj *models.CommandInfo) (string, error) {
	return strconv.FormatInt(obj.Count, 10), nil
}

func (r *commandInfoResolver) LastUsed(ctx context.Context, obj *models.CommandInfo) (*time.Time, error) {
	return obj.LastUsed.Ptr(), nil
}

func (r *commandInfoResolver) Channel(ctx context.Context, obj *models.CommandInfo) (*models.Channel, error) {
	tx := txFromContext(ctx)
	return obj.Channel().One(ctx, tx)
}

func (r *commandInfoResolver) CommandList(ctx context.Context, obj *models.CommandInfo) (*models.CommandList, error) {
	if !obj.CommandListID.Valid {
		return nil, nil //nolint:nilnil
	}
	tx := txFromContext(ctx)
	return obj.CommandList().One(ctx, tx)
}

func (r *commandInfoResolver) CustomCommand(ctx context.Context, obj *models.CommandInfo) (*models.CustomCommand, error) {
	if !obj.CustomCommandID.Valid {
		return nil, nil //nolint:nilnil
	}
	tx := txFromContext(ctx)
	return obj.CustomCommand().One(ctx, tx)
}

func (r *commandInfoResolver) RepeatedCommand(ctx context.Context, obj *models.CommandInfo) (*models.RepeatedCommand, error) {
	tx := txFromContext(ctx)
	rc, err := obj.RepeatedCommand().One(ctx, tx)
	if err == sql.ErrNoRows {
		return nil, nil //nolint:nilnil
	}
	return rc, err
}

func (r *commandInfoResolver) ScheduledCommand(ctx context.Context, obj *models.CommandInfo) (*models.ScheduledCommand, error) {
	tx := txFromContext(ctx)
	sc, err := obj.ScheduledCommand().One(ctx, tx)
	if err == sql.ErrNoRows {
		return nil, nil //nolint:nilnil
	}
	return sc, err
}

func (r *commandListResolver) Items(ctx context.Context, obj *models.CommandList) ([]string, error) {
	return obj.Items, nil
}

func (r *commandListResolver) Channel(ctx context.Context, obj *models.CommandList) (*models.Channel, error) {
	tx := txFromContext(ctx)
	return obj.Channel().One(ctx, tx)
}

func (r *commandListResolver) CommandInfo(ctx context.Context, obj *models.CommandList) (*models.CommandInfo, error) {
	tx := txFromContext(ctx)
	return obj.CommandInfo().One(ctx, tx)
}

func (r *customCommandResolver) Channel(ctx context.Context, obj *models.CustomCommand) (*models.Channel, error) {
	tx := txFromContext(ctx)
	return obj.Channel().One(ctx, tx)
}

func (r *customCommandResolver) CommandInfo(ctx context.Context, obj *models.CustomCommand) (*models.CommandInfo, error) {
	tx := txFromContext(ctx)
	return obj.CommandInfo().One(ctx, tx)
}

func (r *highlightResolver) StartedAt(ctx context.Context, obj *models.Highlight) (*time.Time, error) {
	return obj.StartedAt.Ptr(), nil
}

func (r *highlightResolver) Channel(ctx context.Context, obj *models.Highlight) (*models.Channel, error) {
	tx := txFromContext(ctx)
	return obj.Channel().One(ctx, tx)
}

func (r *queryResolver) ChannelByName(ctx context.Context, name string) (*models.Channel, error) {
	tx := txFromContext(ctx)
	return models.Channels(models.ChannelWhere.Name.EQ(name)).One(ctx, tx)
}

func (r *queryResolver) ChannelByTwitchID(ctx context.Context, twitchID int64) (*models.Channel, error) {
	tx := txFromContext(ctx)
	return models.Channels(models.ChannelWhere.TwitchID.EQ(twitchID)).One(ctx, tx)
}

func (r *quoteResolver) Channel(ctx context.Context, obj *models.Quote) (*models.Channel, error) {
	tx := txFromContext(ctx)
	return obj.Channel().One(ctx, tx)
}

func (r *repeatedCommandResolver) LastCount(ctx context.Context, obj *models.RepeatedCommand) (string, error) {
	return strconv.FormatInt(obj.LastCount, 10), nil
}

func (r *repeatedCommandResolver) InitTimestamp(ctx context.Context, obj *models.RepeatedCommand) (*time.Time, error) {
	return obj.InitTimestamp.Ptr(), nil
}

func (r *repeatedCommandResolver) Channel(ctx context.Context, obj *models.RepeatedCommand) (*models.Channel, error) {
	tx := txFromContext(ctx)
	return obj.Channel().One(ctx, tx)
}

func (r *repeatedCommandResolver) CommandInfo(ctx context.Context, obj *models.RepeatedCommand) (*models.CommandInfo, error) {
	tx := txFromContext(ctx)
	return obj.CommandInfo().One(ctx, tx)
}

func (r *scheduledCommandResolver) LastCount(ctx context.Context, obj *models.ScheduledCommand) (string, error) {
	return strconv.FormatInt(obj.LastCount, 10), nil
}

func (r *scheduledCommandResolver) Channel(ctx context.Context, obj *models.ScheduledCommand) (*models.Channel, error) {
	tx := txFromContext(ctx)
	return obj.Channel().One(ctx, tx)
}

func (r *scheduledCommandResolver) CommandInfo(ctx context.Context, obj *models.ScheduledCommand) (*models.CommandInfo, error) {
	tx := txFromContext(ctx)
	return obj.CommandInfo().One(ctx, tx)
}

func (r *variableResolver) Channel(ctx context.Context, obj *models.Variable) (*models.Channel, error) {
	tx := txFromContext(ctx)
	return obj.Channel().One(ctx, tx)
}

// Autoreply returns generated.AutoreplyResolver implementation.
func (r *Resolver) Autoreply() generated.AutoreplyResolver { return &autoreplyResolver{r} }

// Channel returns generated.ChannelResolver implementation.
func (r *Resolver) Channel() generated.ChannelResolver { return &channelResolver{r} }

// CommandInfo returns generated.CommandInfoResolver implementation.
func (r *Resolver) CommandInfo() generated.CommandInfoResolver { return &commandInfoResolver{r} }

// CommandList returns generated.CommandListResolver implementation.
func (r *Resolver) CommandList() generated.CommandListResolver { return &commandListResolver{r} }

// CustomCommand returns generated.CustomCommandResolver implementation.
func (r *Resolver) CustomCommand() generated.CustomCommandResolver { return &customCommandResolver{r} }

// Highlight returns generated.HighlightResolver implementation.
func (r *Resolver) Highlight() generated.HighlightResolver { return &highlightResolver{r} }

// Query returns generated.QueryResolver implementation.
func (r *Resolver) Query() generated.QueryResolver { return &queryResolver{r} }

// Quote returns generated.QuoteResolver implementation.
func (r *Resolver) Quote() generated.QuoteResolver { return &quoteResolver{r} }

// RepeatedCommand returns generated.RepeatedCommandResolver implementation.
func (r *Resolver) RepeatedCommand() generated.RepeatedCommandResolver {
	return &repeatedCommandResolver{r}
}

// ScheduledCommand returns generated.ScheduledCommandResolver implementation.
func (r *Resolver) ScheduledCommand() generated.ScheduledCommandResolver {
	return &scheduledCommandResolver{r}
}

// Variable returns generated.VariableResolver implementation.
func (r *Resolver) Variable() generated.VariableResolver { return &variableResolver{r} }

type autoreplyResolver struct{ *Resolver }
type channelResolver struct{ *Resolver }
type commandInfoResolver struct{ *Resolver }
type commandListResolver struct{ *Resolver }
type customCommandResolver struct{ *Resolver }
type highlightResolver struct{ *Resolver }
type queryResolver struct{ *Resolver }
type quoteResolver struct{ *Resolver }
type repeatedCommandResolver struct{ *Resolver }
type scheduledCommandResolver struct{ *Resolver }
type variableResolver struct{ *Resolver }
