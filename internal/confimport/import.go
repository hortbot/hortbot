// Package confimport implements importing and exporting of full channel configurations.
package confimport

import (
	"context"

	"github.com/friendsofgo/errors"
	"github.com/hortbot/hortbot/internal/db/models"
	"github.com/volatiletech/null"
	"github.com/volatiletech/sqlboiler/boil"
	"go.opencensus.io/trace"
)

type Config struct {
	Channel     *models.Channel     `json:"channel"`
	Quotes      []*models.Quote     `json:"quotes"`
	Commands    []*Command          `json:"commands"`
	Autoreplies []*models.Autoreply `json:"autoreplies"`
	Variables   []*models.Variable  `json:"variables"`
}

type Command struct {
	Info          *models.CommandInfo      `json:"info"`
	CustomCommand *models.CustomCommand    `json:"custom_command"`
	CommandList   *models.CommandList      `json:"command_list"`
	Repeat        *models.RepeatedCommand  `json:"repeat"`
	Schedule      *models.ScheduledCommand `json:"schedule"`
}

// Insert inserts a config into the database. All IDs will be zero'd before
// inserting, to ensure all inserted rows have new IDs.
func (c *Config) Insert(ctx context.Context, exec boil.ContextExecutor) error {
	ctx, span := trace.StartSpan(ctx, "confimport.Insert")
	defer span.End()

	c.Channel.ID = 0

	if err := c.Channel.Insert(ctx, exec, boil.Infer()); err != nil {
		return errors.Wrap(err, "inserting channel")
	}

	for _, quote := range c.Quotes {
		quote.ChannelID = c.Channel.ID

		if err := quote.Insert(ctx, exec, boil.Infer()); err != nil {
			return errors.Wrap(err, "inserting quote")
		}
	}

	for _, command := range c.Commands {
		command.Info.ID = 0
		command.Info.ChannelID = c.Channel.ID

		if cc := command.CustomCommand; cc != nil {
			cc.ID = 0
			cc.ChannelID = c.Channel.ID

			if err := cc.Insert(ctx, exec, boil.Infer()); err != nil {
				return errors.Wrap(err, "inserting custom command")
			}

			command.Info.CustomCommandID = null.Int64From(cc.ID)
		}

		if cl := command.CommandList; cl != nil {
			cl.ID = 0
			cl.ChannelID = c.Channel.ID

			if err := cl.Insert(ctx, exec, boil.Infer()); err != nil {
				return errors.Wrap(err, "inserting command list")
			}

			command.Info.CommandListID = null.Int64From(cl.ID)
		}

		if err := command.Info.Insert(ctx, exec, boil.Infer()); err != nil {
			return errors.Wrap(err, "inserting command info")
		}

		if r := command.Repeat; r != nil {
			r.ID = 0
			r.ChannelID = c.Channel.ID
			r.CommandInfoID = command.Info.ID

			if err := r.Insert(ctx, exec, boil.Infer()); err != nil {
				return errors.Wrap(err, "inserting repeated command")
			}
		}

		if s := command.Schedule; s != nil {
			s.ID = 0
			s.ChannelID = c.Channel.ID
			s.CommandInfoID = command.Info.ID

			if err := s.Insert(ctx, exec, boil.Infer()); err != nil {
				return errors.Wrap(err, "inserting scheduled command")
			}
		}
	}

	for _, autoreply := range c.Autoreplies {
		autoreply.ID = 0
		autoreply.ChannelID = c.Channel.ID

		if err := autoreply.Insert(ctx, exec, boil.Infer()); err != nil {
			return errors.Wrap(err, "inserting autoreply")
		}
	}

	for _, variable := range c.Variables {
		variable.ID = 0
		variable.ChannelID = c.Channel.ID

		if err := variable.Insert(ctx, exec, boil.Infer()); err != nil {
			return errors.Wrap(err, "inserting variable")
		}
	}

	return nil
}
