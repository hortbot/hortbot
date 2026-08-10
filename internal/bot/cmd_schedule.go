package bot

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/hortbot/hortbot/internal/db/dbsql"
	"github.com/hortbot/hortbot/internal/pkg/must"
	"github.com/hortbot/hortbot/internal/pkg/repeat"
	"github.com/jackc/pgx/v5"
)

var scheduleCommands = newHandlerMap(map[string]handlerFunc{
	"add":    {fn: cmdScheduleAdd, minLevel: AccessLevelModerator},
	"delete": {fn: cmdScheduleDelete, minLevel: AccessLevelModerator},
	"remove": {fn: cmdScheduleDelete, minLevel: AccessLevelModerator},
	"on":     {fn: cmdScheduleOnOff, minLevel: AccessLevelModerator},
	"off":    {fn: cmdScheduleOnOff, minLevel: AccessLevelModerator},
	"list":   {fn: cmdScheduleList, minLevel: AccessLevelModerator},
})

func cmdSchedule(ctx context.Context, s *session, cmd string, args string) error {
	subcommand, args := splitSpace(args)
	subcommand = strings.ToLower(subcommand)

	ok, err := scheduleCommands.Run(ctx, s, subcommand, args)
	if err != nil {
		return err
	}

	if !ok {
		return s.ReplyUsage(ctx, "add|delete|on|off|list ...")
	}

	return nil
}

func cmdScheduleAdd(ctx context.Context, s *session, cmd string, args string) error {
	usage := func() error {
		return s.ReplyUsage(ctx, "<name> <pattern> [message difference]")
	}

	name, args := splitSpace(args)
	pattern, messageDiffStr := splitSpace(args)
	name = cleanCommandName(name)

	if name == "" || pattern == "" {
		return usage()
	}

	pattern = strings.ReplaceAll(pattern, "_", " ")

	expr, err := repeat.ParseCron(pattern)
	if err != nil {
		return s.Replyf(ctx, "Bad cron expression: %s", pattern)
	}

	messageDiff := int64(1)

	if messageDiffStr != "" {
		messageDiff, err = strconv.ParseInt(messageDiffStr, 10, 64)
		if err != nil {
			return usage()
		}

		if messageDiff <= 0 {
			return s.Reply(ctx, "Message difference must be at least 1.")
		}
	}

	info, scheduled, err := findScheduledCommand(ctx, name, s)
	if err != nil {
		return err
	}

	if info == nil {
		return s.Replyf(ctx, "Command '%s' does not exist.", name)
	}

	if !s.UserLevel.CanAccessPG(info.AccessLevel) {
		al := pluralAccessLevel(info.AccessLevel)
		return s.Replyf(ctx, "Command '%s' is restricted to %s; only %s and above can modify its schedule.", name, al, al)
	}

	if scheduled != nil {
		scheduled.CronExpression = pattern
		scheduled.MessageDiff = messageDiff
		scheduled.Enabled = true
		scheduled.LastCount = s.Channel.MessageCount
		scheduled.Editor = s.User

		updated, err := s.Queries.UpdateScheduledCommand(ctx, dbsql.UpdateScheduledCommandParams{
			Enabled: scheduled.Enabled, CronExpression: scheduled.CronExpression,
			MessageDiff: scheduled.MessageDiff, LastCount: scheduled.LastCount,
			Editor: scheduled.Editor, Now: dbsql.TimestamptzFrom(time.Now()), ID: scheduled.ID,
		})
		if err != nil {
			return fmt.Errorf("updating scheduled command: %w", err)
		}
		scheduled.UpdatedAt = updated.UpdatedAt
	} else {
		inserted, err := s.Queries.InsertScheduledCommand(ctx, dbsql.InsertScheduledCommandParams{
			Now:            dbsql.TimestamptzFrom(time.Now()),
			ChannelID:      s.Channel.ID,
			CommandInfoID:  info.ID,
			CronExpression: pattern,
			MessageDiff:    messageDiff,
			LastCount:      s.Channel.MessageCount,
			Creator:        s.User,
			Editor:         s.User,
		})
		if err != nil {
			return fmt.Errorf("inserting scheduled command: %w", err)
		}
		scheduled = new(inserted)
	}

	if err := s.Deps.AddScheduled(ctx, scheduled.ID, expr); err != nil {
		return err
	}

	dUnit := "message has passed."
	if messageDiff != 1 {
		dUnit = "messages have passed."
	}

	return s.Replyf(ctx, "Command '%s' has been scheduled with '%s' and will run if at least %d %s", name, pattern, messageDiff, dUnit)
}

func cmdScheduleDelete(ctx context.Context, s *session, cmd string, args string) error {
	name, _ := splitSpace(args)
	name = cleanCommandName(name)

	if name == "" {
		return s.ReplyUsage(ctx, "<name>")
	}

	info, scheduled, err := findScheduledCommand(ctx, name, s)
	if err != nil {
		return err
	}

	if info == nil {
		return s.Replyf(ctx, "Command '%s' does not exist.", name)
	}

	if scheduled == nil {
		return s.Replyf(ctx, "Command '%s' has no schedule.", name)
	}

	if !s.UserLevel.CanAccessPG(info.AccessLevel) {
		al := pluralAccessLevel(info.AccessLevel)
		return s.Replyf(ctx, "Command '%s' is restricted to %s; only %s and above can modify its schedule.", name, al, al)
	}

	if err := s.Queries.DeleteScheduledCommand(ctx, scheduled.ID); err != nil {
		return fmt.Errorf("deleting scheduled command: %w", err)
	}

	if err := s.Deps.RemoveScheduled(ctx, scheduled.ID); err != nil {
		return err
	}

	return s.Replyf(ctx, "Command '%s' is no longer scheduled.", name)
}

func cmdScheduleOnOff(ctx context.Context, s *session, cmd string, args string) error {
	name, _ := splitSpace(args)
	name = cleanCommandName(name)

	if name == "" {
		return s.ReplyUsage(ctx, "<name>")
	}

	enable := cmd == "on"

	info, scheduled, err := findScheduledCommand(ctx, name, s)
	if err != nil {
		return err
	}

	if info == nil {
		return s.Replyf(ctx, "Command '%s' does not exist.", name)
	}

	if scheduled == nil {
		return s.Replyf(ctx, "Command '%s' has no schedule.", name)
	}

	if !s.UserLevel.CanAccessPG(info.AccessLevel) {
		al := pluralAccessLevel(info.AccessLevel)
		return s.Replyf(ctx, "Command '%s' is restricted to %s; only %s and above can modify its schedule.", name, al, al)
	}

	if scheduled.Enabled == enable {
		if enable {
			return s.Replyf(ctx, "Scheduled command '%s' is already enabled.", name)
		}
		return s.Replyf(ctx, "Scheduled command '%s' is already disabled.", name)
	}

	scheduled.Enabled = enable
	scheduled.LastCount = s.Channel.MessageCount
	scheduled.Editor = s.User

	updated, err := s.Queries.UpdateScheduledCommand(ctx, dbsql.UpdateScheduledCommandParams{
		Enabled: scheduled.Enabled, CronExpression: scheduled.CronExpression,
		MessageDiff: scheduled.MessageDiff, LastCount: scheduled.LastCount,
		Editor: scheduled.Editor, Now: dbsql.TimestamptzFrom(time.Now()), ID: scheduled.ID,
	})
	if err != nil {
		return fmt.Errorf("updating scheduled command: %w", err)
	}
	scheduled.UpdatedAt = updated.UpdatedAt

	expr := must.Must(repeat.ParseCron(scheduled.CronExpression))

	if enable {
		err = s.Deps.AddScheduled(ctx, scheduled.ID, expr)
	} else {
		err = s.Deps.RemoveScheduled(ctx, scheduled.ID)
	}

	if err != nil {
		return err
	}

	if enable {
		return s.Replyf(ctx, "Scheduled command '%s' is now enabled.", name)
	}

	return s.Replyf(ctx, "Scheduled command '%s' is now disabled.", name)
}

func cmdScheduleList(ctx context.Context, s *session, cmd string, args string) error {
	scheduleds, err := s.Queries.ListScheduledCommandsWithNames(ctx, s.Channel.ID)
	if err != nil {
		return fmt.Errorf("getting scheduled commands: %w", err)
	}

	if len(scheduleds) == 0 {
		return s.Reply(ctx, "There are no scheduled commands.")
	}

	var builder strings.Builder

	builder.WriteString("Scheduled commands: ")
	for i, scheduled := range scheduleds {
		if i != 0 {
			builder.WriteString(", ")
		}

		builder.WriteString(scheduled.Name)
		builder.WriteString(" [")

		if scheduled.Enabled {
			builder.WriteString("ON")
		} else {
			builder.WriteString("OFF")
		}

		builder.WriteString("] (")
		builder.WriteString(scheduled.CronExpression)
		builder.WriteByte(')')
	}

	return s.Reply(ctx, builder.String())
}

func findScheduledCommand(ctx context.Context, name string, s *session) (*dbsql.CommandInfo, *dbsql.ScheduledCommand, error) {
	info, _, found, err := s.Queries.LookupCommand(ctx, s.Channel.ID, name, true)
	if err != nil {
		return nil, nil, fmt.Errorf("getting command info: %w", err)
	}
	if !found {
		return nil, nil, nil
	}
	scheduled, err := s.Queries.GetScheduledCommandByInfo(ctx, info.ID)
	if err != nil && errors.Is(err, pgx.ErrNoRows) {
		return info, nil, nil
	}
	if err != nil {
		return nil, nil, fmt.Errorf("getting scheduled command: %w", err)
	}
	return info, new(scheduled), nil
}
