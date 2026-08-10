// Package confimport implements importing and exporting of full channel configurations.
package confimport

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/hortbot/hortbot/internal/db/dbsql"
	"github.com/jackc/pgx/v5/pgtype"
)

// Config is a channel's full configuration, serialized.
type Config struct {
	Channel     *Channel     `json:"channel"`
	Quotes      []*Quote     `json:"quotes"`
	Commands    []*Command   `json:"commands"`
	Autoreplies []*Autoreply `json:"autoreplies"`
	Variables   []*Variable  `json:"variables"`
}

// Command is a single command, including all variants and repeats/schedules.
type Command struct {
	Info          *CommandInfo      `json:"info"`
	CustomCommand *CustomCommand    `json:"custom_command"`
	CommandList   *CommandList      `json:"command_list"`
	Repeat        *RepeatedCommand  `json:"repeat"`
	Schedule      *ScheduledCommand `json:"schedule"`
}

type importInserter func(context.Context, []byte) (int64, error)

// Insert inserts a config into the database. All IDs are replaced with newly
// allocated identity values while preserving the serialized timestamps.
func (c *Config) Insert(ctx context.Context, queries *dbsql.Queries) error {
	if err := c.validate(); err != nil {
		return err
	}
	c.applyImportDefaults(time.Now())

	id, err := insertImported(ctx, c.Channel, queries.InsertImportedChannel, "channel")
	if err != nil {
		return err
	}
	c.Channel.ID = id

	for _, quote := range c.Quotes {
		quote.ChannelID = id
		quote.ID, err = insertImported(ctx, quote, queries.InsertImportedQuote, "quote")
		if err != nil {
			return err
		}
	}

	for _, command := range c.Commands {
		command.Info.ChannelID = id
		if value := command.CustomCommand; value != nil {
			value.ChannelID = id
			value.ID, err = insertImported(ctx, value, queries.InsertImportedCustomCommand, "custom command")
			if err != nil {
				return err
			}
			command.Info.CustomCommandID = dbsql.Int8From(value.ID)
			command.Info.CommandListID = pgtype.Int8{}
		}
		if value := command.CommandList; value != nil {
			value.ChannelID = id
			value.ID, err = insertImported(ctx, value, queries.InsertImportedCommandList, "command list")
			if err != nil {
				return err
			}
			command.Info.CommandListID = dbsql.Int8From(value.ID)
			command.Info.CustomCommandID = pgtype.Int8{}
		}
		command.Info.ID, err = insertImported(ctx, command.Info, queries.InsertImportedCommandInfo, "command info")
		if err != nil {
			return err
		}
		if value := command.Repeat; value != nil {
			value.ChannelID = id
			value.CommandInfoID = command.Info.ID
			value.ID, err = insertImported(ctx, value, queries.InsertImportedRepeatedCommand, "repeated command")
			if err != nil {
				return err
			}
		}
		if value := command.Schedule; value != nil {
			value.ChannelID = id
			value.CommandInfoID = command.Info.ID
			value.ID, err = insertImported(ctx, value, queries.InsertImportedScheduledCommand, "scheduled command")
			if err != nil {
				return err
			}
		}
	}

	for _, autoreply := range c.Autoreplies {
		autoreply.ChannelID = id
		autoreply.ID, err = insertImported(ctx, autoreply, queries.InsertImportedAutoreply, "autoreply")
		if err != nil {
			return err
		}
	}
	for _, variable := range c.Variables {
		variable.ChannelID = id
		variable.ID, err = insertImported(ctx, variable, queries.InsertImportedVariable, "variable")
		if err != nil {
			return err
		}
	}
	return nil
}

func (c *Config) validate() error {
	if c.Channel == nil {
		return errors.New("config has no channel")
	}
	for i, quote := range c.Quotes {
		if quote == nil {
			return fmt.Errorf("quote %d is null", i)
		}
	}
	for i, command := range c.Commands {
		if command == nil {
			return fmt.Errorf("command %d is null", i)
		}
		if command.Info == nil {
			return fmt.Errorf("command %d has no info", i)
		}
		if (command.CustomCommand == nil) == (command.CommandList == nil) {
			return fmt.Errorf("command %d must have exactly one implementation", i)
		}
	}
	for i, autoreply := range c.Autoreplies {
		if autoreply == nil {
			return fmt.Errorf("autoreply %d is null", i)
		}
	}
	for i, variable := range c.Variables {
		if variable == nil {
			return fmt.Errorf("variable %d is null", i)
		}
	}
	return nil
}

func (c *Config) applyImportDefaults(now time.Time) {
	defaultTimestamps(&c.Channel.CreatedAt, &c.Channel.UpdatedAt, now)
	defaultTimestamp(&c.Channel.LastSeen, now)
	defaultStrings(&c.Channel.Ignored)
	defaultStrings(&c.Channel.CustomOwners)
	defaultStrings(&c.Channel.CustomMods)
	defaultStrings(&c.Channel.CustomRegulars)
	defaultStrings(&c.Channel.PermittedLinks)
	defaultStrings(&c.Channel.FilterBannedPhrasesPatterns)
	if c.Channel.FilterExemptLevel == "" {
		c.Channel.FilterExemptLevel = dbsql.AccessLevelSubscriber
	}

	for _, quote := range c.Quotes {
		defaultTimestamps(&quote.CreatedAt, &quote.UpdatedAt, now)
	}
	for _, command := range c.Commands {
		defaultTimestamps(&command.Info.CreatedAt, &command.Info.UpdatedAt, now)
		if value := command.CustomCommand; value != nil {
			defaultTimestamps(&value.CreatedAt, &value.UpdatedAt, now)
		}
		if value := command.CommandList; value != nil {
			defaultTimestamps(&value.CreatedAt, &value.UpdatedAt, now)
			defaultStrings(&value.Items)
		}
		if value := command.Repeat; value != nil {
			defaultTimestamps(&value.CreatedAt, &value.UpdatedAt, now)
			if value.MessageDiff == 0 {
				value.MessageDiff = 1
			}
		}
		if value := command.Schedule; value != nil {
			defaultTimestamps(&value.CreatedAt, &value.UpdatedAt, now)
			if value.MessageDiff == 0 {
				value.MessageDiff = 1
			}
		}
	}
	for _, autoreply := range c.Autoreplies {
		defaultTimestamps(&autoreply.CreatedAt, &autoreply.UpdatedAt, now)
	}
	for _, variable := range c.Variables {
		defaultTimestamps(&variable.CreatedAt, &variable.UpdatedAt, now)
	}
}

func defaultTimestamps(createdAt, updatedAt *pgtype.Timestamptz, now time.Time) {
	defaultTimestamp(createdAt, now)
	defaultTimestamp(updatedAt, now)
}

func defaultTimestamp(value *pgtype.Timestamptz, now time.Time) {
	if !value.Valid || value.Time.IsZero() {
		*value = dbsql.TimestamptzFrom(now)
	}
}

func defaultStrings(value *[]string) {
	if *value == nil {
		*value = []string{}
	}
}

func insertImported(ctx context.Context, model any, insert importInserter, label string) (int64, error) {
	data, err := json.Marshal(model)
	if err != nil {
		return 0, fmt.Errorf("marshaling %s: %w", label, err)
	}
	id, err := insert(ctx, data)
	if err != nil {
		return 0, fmt.Errorf("inserting %s: %w", label, err)
	}
	return id, nil
}
