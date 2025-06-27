// Package confimport implements importing and exporting of full channel configurations.
package confimport

import (
	"context"
	"fmt"

	"github.com/aarondl/null/v8"
	"github.com/aarondl/sqlboiler/v4/boil"
	"github.com/hortbot/hortbot/internal/db/models"
)

// Config is a channel's full configuration, serialized.
type Config struct {
	Channel     *models.Channel     `json:"channel"`
	Quotes      []*models.Quote     `json:"quotes"`
	Commands    []*Command          `json:"commands"`
	Autoreplies []*models.Autoreply `json:"autoreplies"`
	Variables   []*models.Variable  `json:"variables"`
}

// Command is a single command, including all variants and repeats/schedules.
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
	c.Channel.ID = 0

	if err := c.Channel.Insert(ctx, exec, boil.Infer()); err != nil {
		return fmt.Errorf("inserting channel: %w", err)
	}

	for _, quote := range c.Quotes {
		quote.ChannelID = c.Channel.ID

		if err := quote.Insert(ctx, exec, boil.Infer()); err != nil {
			return fmt.Errorf("inserting quote: %w", err)
		}
	}

	for _, command := range c.Commands {
		command.Info.ID = 0
		command.Info.ChannelID = c.Channel.ID

		if cc := command.CustomCommand; cc != nil {
			cc.ID = 0
			cc.ChannelID = c.Channel.ID

			if err := cc.Insert(ctx, exec, boil.Infer()); err != nil {
				return fmt.Errorf("inserting custom command: %w", err)
			}

			command.Info.CustomCommandID = null.Int64From(cc.ID)
		}

		if cl := command.CommandList; cl != nil {
			cl.ID = 0
			cl.ChannelID = c.Channel.ID

			if err := cl.Insert(ctx, exec, boil.Infer()); err != nil {
				return fmt.Errorf("inserting command list: %w", err)
			}

			command.Info.CommandListID = null.Int64From(cl.ID)
		}

		if err := command.Info.Insert(ctx, exec, boil.Infer()); err != nil {
			return fmt.Errorf("inserting command info: %w", err)
		}

		if r := command.Repeat; r != nil {
			r.ID = 0
			r.ChannelID = c.Channel.ID
			r.CommandInfoID = command.Info.ID

			if err := r.Insert(ctx, exec, boil.Infer()); err != nil {
				return fmt.Errorf("inserting repeated command: %w", err)
			}
		}

		if s := command.Schedule; s != nil {
			s.ID = 0
			s.ChannelID = c.Channel.ID
			s.CommandInfoID = command.Info.ID

			if err := s.Insert(ctx, exec, boil.Infer()); err != nil {
				return fmt.Errorf("inserting scheduled command: %w", err)
			}
		}
	}

	for _, autoreply := range c.Autoreplies {
		autoreply.ID = 0
		autoreply.ChannelID = c.Channel.ID

		if err := autoreply.Insert(ctx, exec, boil.Infer()); err != nil {
			return fmt.Errorf("inserting autoreply: %w", err)
		}
	}

	for _, variable := range c.Variables {
		variable.ID = 0
		variable.ChannelID = c.Channel.ID

		if err := variable.Insert(ctx, exec, boil.Infer()); err != nil {
			return fmt.Errorf("inserting variable: %w", err)
		}
	}

	return nil
}
