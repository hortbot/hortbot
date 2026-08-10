package confimport

import (
	"context"
	"fmt"

	"github.com/hortbot/hortbot/internal/db/dbsql"
)

// ExportByName exports a channel's full configuration, keyed on channel name.
func ExportByName(ctx context.Context, queries *dbsql.Queries, name string) (*Config, error) {
	channelRow, err := queries.GetChannelByName(ctx, name)
	if err != nil {
		return nil, fmt.Errorf("getting channel: %w", err)
	}
	channel := Channel(channelRow)

	quoteRows, err := queries.ListQuotes(ctx, channel.ID)
	if err != nil {
		return nil, fmt.Errorf("getting quotes: %w", err)
	}

	autoreplyRows, err := queries.ListAutoreplies(ctx, channel.ID)
	if err != nil {
		return nil, fmt.Errorf("getting autoreplies: %w", err)
	}
	autoreplies := make([]*Autoreply, len(autoreplyRows))
	for i := range autoreplyRows {
		autoreplies[i] = (*Autoreply)(&autoreplyRows[i])
	}

	variableRows, err := queries.ListVariables(ctx, channel.ID)
	if err != nil {
		return nil, fmt.Errorf("getting variables: %w", err)
	}

	infoRows, err := queries.ListCommandInfos(ctx, channel.ID)
	if err != nil {
		return nil, fmt.Errorf("getting command infos: %w", err)
	}
	customCommandRows, err := queries.ListCustomCommands(ctx, channel.ID)
	if err != nil {
		return nil, fmt.Errorf("getting custom commands: %w", err)
	}
	commandListRows, err := queries.ListCommandLists(ctx, channel.ID)
	if err != nil {
		return nil, fmt.Errorf("getting command lists: %w", err)
	}
	repeatedCommandRows, err := queries.ListRepeatedCommands(ctx, channel.ID)
	if err != nil {
		return nil, fmt.Errorf("getting repeated commands: %w", err)
	}
	scheduledCommandRows, err := queries.ListScheduledCommands(ctx, channel.ID)
	if err != nil {
		return nil, fmt.Errorf("getting scheduled commands: %w", err)
	}

	commands, err := assembleCommands(
		infoRows, customCommandRows, commandListRows,
		repeatedCommandRows, scheduledCommandRows,
	)
	if err != nil {
		return nil, err
	}

	return &Config{
		Channel: &channel, Quotes: pointers(quoteRows), Commands: commands,
		Autoreplies: autoreplies, Variables: pointers(variableRows),
	}, nil
}

func assembleCommands(
	infoRows []dbsql.CommandInfo,
	customCommandRows []dbsql.CustomCommand,
	commandListRows []dbsql.CommandList,
	repeatedCommandRows []dbsql.RepeatedCommand,
	scheduledCommandRows []dbsql.ScheduledCommand,
) ([]*Command, error) {
	customCommands := make(map[int64]*CustomCommand, len(customCommandRows))
	for i := range customCommandRows {
		customCommands[customCommandRows[i].ID] = &customCommandRows[i]
	}
	commandLists := make(map[int64]*CommandList, len(commandListRows))
	for i := range commandListRows {
		commandLists[commandListRows[i].ID] = &commandListRows[i]
	}
	repeatedCommands := make(map[int64]*RepeatedCommand, len(repeatedCommandRows))
	for i := range repeatedCommandRows {
		repeatedCommands[repeatedCommandRows[i].CommandInfoID] = (*RepeatedCommand)(&repeatedCommandRows[i])
	}
	scheduledCommands := make(map[int64]*ScheduledCommand, len(scheduledCommandRows))
	for i := range scheduledCommandRows {
		scheduledCommands[scheduledCommandRows[i].CommandInfoID] = &scheduledCommandRows[i]
	}

	commands := make([]*Command, len(infoRows))
	for i := range infoRows {
		info := (*CommandInfo)(&infoRows[i])
		command := &Command{
			Info:     info,
			Repeat:   repeatedCommands[info.ID],
			Schedule: scheduledCommands[info.ID],
		}
		switch {
		case info.CustomCommandID.Valid:
			command.CustomCommand = customCommands[info.CustomCommandID.Int64]
			if command.CustomCommand == nil {
				return nil, fmt.Errorf("command %q references missing custom command %d", info.Name, info.CustomCommandID.Int64)
			}
		case info.CommandListID.Valid:
			command.CommandList = commandLists[info.CommandListID.Int64]
			if command.CommandList == nil {
				return nil, fmt.Errorf("command %q references missing command list %d", info.Name, info.CommandListID.Int64)
			}
		default:
			return nil, fmt.Errorf("command %q has no implementation", info.Name)
		}
		commands[i] = command
	}
	return commands, nil
}

func pointers[T any](values []T) []*T {
	out := make([]*T, len(values))
	for i := range values {
		out[i] = &values[i]
	}
	return out
}
