package bot

import (
	"context"
	"database/sql"
	"sort"
	"strconv"
	"strings"

	"github.com/gobuffalo/flect"
	"github.com/hortbot/hortbot/internal/db/models"
	"github.com/hortbot/hortbot/internal/pkg/repeat"
	"github.com/volatiletech/sqlboiler/boil"
	"github.com/volatiletech/sqlboiler/queries/qm"
)

var scheduleCommands = newHandlerMap(map[string]handlerFunc{
	"add":    {fn: cmdScheduleAdd, minLevel: levelModerator},
	"delete": {fn: cmdScheduleDelete, minLevel: levelModerator},
	"remove": {fn: cmdScheduleDelete, minLevel: levelModerator},
	"on":     {fn: cmdScheduleOnOff, minLevel: levelModerator},
	"off":    {fn: cmdScheduleOnOff, minLevel: levelModerator},
	"list":   {fn: cmdScheduleList, minLevel: levelModerator},
})

func cmdSchedule(ctx context.Context, s *session, cmd string, args string) error {
	subcommand, args := splitSpace(args)
	subcommand = strings.ToLower(subcommand)

	ok, err := scheduleCommands.run(ctx, s, subcommand, args)
	if err != nil {
		return err
	}

	if !ok {
		return s.ReplyUsage("add|delete|on|off|list ...")
	}

	return nil
}

func cmdScheduleAdd(ctx context.Context, s *session, cmd string, args string) error {
	usage := func() error {
		return s.ReplyUsage("<name> <pattern> [message difference]")
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
		return s.Replyf("Bad cron expression: %s", pattern)
	}

	messageDiff := int64(1)

	if messageDiffStr != "" {
		messageDiff, err = strconv.ParseInt(messageDiffStr, 10, 64)
		if err != nil {
			return usage()
		}

		if messageDiff <= 0 {
			return s.Reply("Message difference must be at least 1.")
		}
	}

	command, scheduled, err := findScheduledCommand(ctx, name, s)
	if err != nil {
		return err
	}

	if command == nil {
		return s.Replyf("Command '%s' does not exist.", name)
	}

	if !s.UserLevel.CanAccess(newAccessLevel(command.AccessLevel)) {
		al := flect.Pluralize(command.AccessLevel)
		return s.Replyf("Command '%s' is restricted to %s; only %s and above can modify its schedule.", name, al, al)
	}

	if scheduled != nil {
		scheduled.CronExpression = pattern
		scheduled.MessageDiff = messageDiff
		scheduled.Enabled = true
		scheduled.LastCount = s.N
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
			return err
		}
	} else {
		scheduled = &models.ScheduledCommand{
			ChannelID:       s.Channel.ID,
			CustomCommandID: command.ID,
			Enabled:         true,
			CronExpression:  pattern,
			MessageDiff:     messageDiff,
			LastCount:       s.N,
			Creator:         s.User,
			Editor:          s.User,
		}

		if err := scheduled.Insert(ctx, s.Tx, boil.Infer()); err != nil {
			return err
		}
	}

	s.Deps.UpdateSchedule(scheduled.ID, true, expr)

	dUnit := "message has passed."
	if messageDiff != 1 {
		dUnit = "messages have passed."
	}

	return s.Replyf("Command '%s' has been scheduled with '%s' and will run if at least %d %s", name, pattern, messageDiff, dUnit)
}

func cmdScheduleDelete(ctx context.Context, s *session, cmd string, args string) error {
	name, _ := splitSpace(args)
	name = cleanCommandName(name)

	if name == "" {
		return s.ReplyUsage("<name>")
	}

	command, scheduled, err := findScheduledCommand(ctx, name, s)
	if err != nil {
		return err
	}

	if command == nil {
		return s.Replyf("Command '%s' does not exist.", name)
	}

	if scheduled == nil {
		return s.Replyf("Command '%s' has no schedule.", name)
	}

	if !s.UserLevel.CanAccess(newAccessLevel(command.AccessLevel)) {
		al := flect.Pluralize(command.AccessLevel)
		return s.Replyf("Command '%s' is restricted to %s; only %s and above can modify its schedule.", name, al, al)
	}

	if err := scheduled.Delete(ctx, s.Tx); err != nil {
		return err
	}

	s.Deps.UpdateSchedule(scheduled.ID, false, nil)

	return s.Replyf("Command '%s' is no longer scheduled.", name)
}

func cmdScheduleOnOff(ctx context.Context, s *session, cmd string, args string) error {
	name, _ := splitSpace(args)
	name = cleanCommandName(name)

	if name == "" {
		return s.ReplyUsage("<name>")
	}

	enable := cmd == "on"

	command, scheduled, err := findScheduledCommand(ctx, name, s)
	if err != nil {
		return err
	}

	if command == nil {
		return s.Replyf("Command '%s' does not exist.", name)
	}

	if scheduled == nil {
		return s.Replyf("Command '%s' has no schedule.", name)
	}

	if !s.UserLevel.CanAccess(newAccessLevel(command.AccessLevel)) {
		al := flect.Pluralize(command.AccessLevel)
		return s.Replyf("Command '%s' is restricted to %s; only %s and above can modify its schedule.", name, al, al)
	}

	if scheduled.Enabled == enable {
		if enable {
			return s.Replyf("Scheduled command '%s' is already enabled.", name)
		}
		return s.Replyf("Scheduled command '%s' is already disabled.", name)
	}

	scheduled.Enabled = enable
	scheduled.LastCount = s.N
	scheduled.Editor = s.User

	columns := boil.Whitelist(
		models.ScheduledCommandColumns.UpdatedAt,
		models.ScheduledCommandColumns.Enabled,
		models.ScheduledCommandColumns.LastCount,
		models.ScheduledCommandColumns.Editor,
	)

	if err := scheduled.Update(ctx, s.Tx, columns); err != nil {
		return err
	}

	expr, err := repeat.ParseCron(scheduled.CronExpression)
	if err != nil {
		panic(err)
	}

	s.Deps.UpdateSchedule(scheduled.ID, enable, expr)

	if enable {
		return s.Replyf("Scheduled command '%s' is now enabled.", name)
	}

	return s.Replyf("Scheduled command '%s' is now disabled.", name)
}

func cmdScheduleList(ctx context.Context, s *session, cmd string, args string) error {
	scheduleds, err := s.Channel.ScheduledCommands(
		qm.Load(models.ScheduledCommandRels.CustomCommand),
	).All(ctx, s.Tx)
	if err != nil {
		return err
	}

	if len(scheduleds) == 0 {
		return s.Reply("There are no scheduled commands.")
	}

	sort.Slice(scheduleds, func(i, j int) bool {
		return scheduleds[i].R.CustomCommand.Name < scheduleds[j].R.CustomCommand.Name
	})

	var builder strings.Builder

	builder.WriteString("Scheduled commands: ")
	for i, scheduled := range scheduleds {
		if i != 0 {
			builder.WriteString(", ")
		}

		builder.WriteString(scheduled.R.CustomCommand.Name)
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

	return s.Reply(builder.String())
}

func findScheduledCommand(ctx context.Context, name string, s *session) (*models.CustomCommand, *models.ScheduledCommand, error) {
	command, err := s.Channel.CustomCommands(
		models.CustomCommandWhere.Name.EQ(name),
		qm.Load(models.CustomCommandRels.ScheduledCommand),
		qm.For("UPDATE"),
	).One(ctx, s.Tx)

	if err == sql.ErrNoRows {
		return nil, nil, nil
	}

	if err != nil {
		return nil, nil, err
	}

	return command, command.R.ScheduledCommand, nil
}
