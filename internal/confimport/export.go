package confimport

import (
	"context"
	"fmt"
	"sort"

	"github.com/hortbot/hortbot/internal/db/models"
	"github.com/volatiletech/sqlboiler/v4/boil"
	"github.com/volatiletech/sqlboiler/v4/queries/qm"
	"go.opencensus.io/trace"
)

// ExportByName exports a channel's full configuration, keyed on channel name.
func ExportByName(ctx context.Context, exec boil.ContextExecutor, name string) (*Config, error) {
	return export(ctx, exec, models.ChannelWhere.Name.EQ(name))
}

func export(ctx context.Context, exec boil.ContextExecutor, mod qm.QueryMod) (*Config, error) {
	ctx, span := trace.StartSpan(ctx, "confimport.export")
	defer span.End()

	channel, err := models.Channels(
		mod,
		qm.Load(models.ChannelRels.Autoreplies),
		qm.Load(models.ChannelRels.CommandInfos),
		qm.Load(qm.Rels(models.ChannelRels.CommandInfos, models.CommandInfoRels.CommandList)),
		qm.Load(qm.Rels(models.ChannelRels.CommandInfos, models.CommandInfoRels.CustomCommand)),
		qm.Load(qm.Rels(models.ChannelRels.CommandInfos, models.CommandInfoRels.RepeatedCommand)),
		qm.Load(qm.Rels(models.ChannelRels.CommandInfos, models.CommandInfoRels.ScheduledCommand)),
		qm.Load(models.ChannelRels.Quotes),
		qm.Load(models.ChannelRels.Variables),
	).One(ctx, exec)
	if err != nil {
		return nil, fmt.Errorf("getting channels: %w", err)
	}

	infos := channel.R.CommandInfos
	commands := make([]*Command, len(infos))

	for i, info := range infos {
		commands[i] = &Command{
			Info:          info,
			CustomCommand: info.R.CustomCommand,
			CommandList:   info.R.CommandList,
			Repeat:        info.R.RepeatedCommand,
			Schedule:      info.R.ScheduledCommand,
		}
	}

	sort.Slice(commands, func(i, j int) bool {
		return commands[i].Info.Name < commands[j].Info.Name
	})

	return &Config{
		Channel:     channel,
		Quotes:      channel.R.Quotes,
		Autoreplies: channel.R.Autoreplies,
		Variables:   channel.R.Variables,
		Commands:    commands,
	}, nil
}
