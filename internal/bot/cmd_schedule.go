package bot

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/aarondl/sqlboiler/v4/boil"
	"github.com/aarondl/sqlboiler/v4/queries/qm"
	"github.com/gobuffalo/flect"
	"github.com/hortbot/hortbot/internal/db/models"
	"github.com/hortbot/hortbot/internal/pkg/must"
	"github.com/hortbot/hortbot/internal/pkg/repeat"
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
		al := flect.Pluralize(info.AccessLevel)
		return s.Replyf(ctx, "Command '%s' is restricted to %s; only %s and above can modify its schedule.", name, al, al)
	}

	if scheduled != nil {
		scheduled.CronExpression = pattern
		scheduled.MessageDiff = messageDiff
		scheduled.Enabled = true
		scheduled.LastCount = s.Channel.MessageCount
		scheduled.Editor = s.User

		columns := boil.Whitelist(
			models.ScheduledCommandColumns.UpdatedAt,
			models.ScheduledCommandColumns.CronExpression,
			models.ScheduledCommandColumns.MessageDiff,
			models.ScheduledCommandColumns.Enabled,
			models.ScheduledCommandColumns.LastCount,
			models.ScheduledCommandColumns.Editor,
		)

		if err := scheduled.Update(ctx, s.Tx, columns); err != nil {
			return fmt.Errorf("updating scheduled command: %w", err)
		}
	} else {
		scheduled = &models.ScheduledCommand{
			ChannelID:      s.Channel.ID,
			CommandInfoID:  info.ID,
			Enabled:        true,
			CronExpression: pattern,
			MessageDiff:    messageDiff,
			LastCount:      s.Channel.MessageCount,
			Creator:        s.User,
			Editor:         s.User,
		}

		if err := scheduled.Insert(ctx, s.Tx, boil.Infer()); err != nil {
			return fmt.Errorf("inserting scheduled command: %w", err)
		}
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
		al := flect.Pluralize(info.AccessLevel)
		return s.Replyf(ctx, "Command '%s' is restricted to %s; only %s and above can modify its schedule.", name, al, al)
	}

	if err := scheduled.Delete(ctx, s.Tx); err != nil {
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
		al := flect.Pluralize(info.AccessLevel)
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

	columns := boil.Whitelist(
		models.ScheduledCommandColumns.UpdatedAt,
		models.ScheduledCommandColumns.Enabled,
		models.ScheduledCommandColumns.LastCount,
		models.ScheduledCommandColumns.Editor,
	)

	if err := scheduled.Update(ctx, s.Tx, columns); err != nil {
		return fmt.Errorf("updating scheduled command: %w", err)
	}

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
	scheduleds, err := s.Channel.ScheduledCommands(
		qm.Load(models.ScheduledCommandRels.CommandInfo),
	).All(ctx, s.Tx)
	if err != nil {
		return fmt.Errorf("getting scheduled commands: %w", err)
	}

	if len(scheduleds) == 0 {
		return s.Reply(ctx, "There are no scheduled commands.")
	}

	slices.SortFunc(scheduleds, func(a, b *models.ScheduledCommand) int {
		return strings.Compare(a.R.CommandInfo.Name, b.R.CommandInfo.Name)
	})

	var builder strings.Builder

	builder.WriteString("Scheduled commands: ")
	for i, scheduled := range scheduleds {
		if i != 0 {
			builder.WriteString(", ")
		}

		builder.WriteString(scheduled.R.CommandInfo.Name)
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

func findScheduledCommand(ctx context.Context, name string, s *session) (*models.CommandInfo, *models.ScheduledCommand, error) {
	info, err := s.Channel.CommandInfos(
		models.CommandInfoWhere.Name.EQ(name),
		qm.Load(models.CommandInfoRels.ScheduledCommand),
		qm.For("UPDATE"),
	).One(ctx, s.Tx)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil, nil
	}

	if err != nil {
		return nil, nil, fmt.Errorf("getting command info: %w", err)
	}

	return info, info.R.ScheduledCommand, nil
}
