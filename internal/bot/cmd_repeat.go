package bot

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/aarondl/null/v8"
	"github.com/aarondl/sqlboiler/v4/boil"
	"github.com/aarondl/sqlboiler/v4/queries/qm"
	"github.com/gobuffalo/flect"
	"github.com/hortbot/hortbot/internal/db/models"
)

var repeatCommands = newHandlerMap(map[string]handlerFunc{
	"add":    {fn: cmdRepeatAdd, minLevel: AccessLevelModerator},
	"delete": {fn: cmdRepeatDelete, minLevel: AccessLevelModerator},
	"remove": {fn: cmdRepeatDelete, minLevel: AccessLevelModerator},
	"on":     {fn: cmdRepeatOnOff, minLevel: AccessLevelModerator},
	"off":    {fn: cmdRepeatOnOff, minLevel: AccessLevelModerator},
	"list":   {fn: cmdRepeatList, minLevel: AccessLevelModerator},
})

func cmdRepeat(ctx context.Context, s *session, cmd string, args string) error {
	subcommand, args := splitSpace(args)
	subcommand = strings.ToLower(subcommand)

	ok, err := repeatCommands.Run(ctx, s, subcommand, args)
	if err != nil {
		return err
	}

	if !ok {
		return s.ReplyUsage(ctx, "add|delete|on|off|list ...")
	}

	return nil
}

func cmdRepeatAdd(ctx context.Context, s *session, cmd string, args string) error {
	usage := func() error {
		return s.ReplyUsage(ctx, "<name> <delay in seconds> [message difference]")
	}

	name, args := splitSpace(args)
	delayStr, messageDiffStr := splitSpace(args)
	name = cleanCommandName(name)

	if name == "" {
		return usage()
	}

	delay, err := strconv.Atoi(delayStr)
	if err != nil {
		return usage()
	}

	if delay < 30 {
		return s.Reply(ctx, "Delay must be at least 30 seconds.")
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

	info, repeat, err := findRepeatedCommand(ctx, name, s)
	if err != nil {
		return err
	}

	if info == nil {
		return s.Replyf(ctx, "Command '%s' does not exist.", name)
	}

	if !s.UserLevel.CanAccessPG(info.AccessLevel) {
		al := flect.Pluralize(info.AccessLevel)
		return s.Replyf(ctx, "Command '%s' is restricted to %s; only %s and above can modify its repeat.", name, al, al)
	}

	if repeat != nil {
		repeat.Delay = delay
		repeat.MessageDiff = messageDiff
		repeat.Enabled = true
		repeat.LastCount = s.Channel.MessageCount
		repeat.Editor = s.User
		repeat.InitTimestamp = null.Time{}

		columns := boil.Whitelist(
			models.RepeatedCommandColumns.UpdatedAt,
			models.RepeatedCommandColumns.Delay,
			models.RepeatedCommandColumns.MessageDiff,
			models.RepeatedCommandColumns.Enabled,
			models.RepeatedCommandColumns.LastCount,
			models.RepeatedCommandColumns.Editor,
			models.RepeatedCommandColumns.InitTimestamp,
		)

		if err := repeat.Update(ctx, s.Tx, columns); err != nil {
			return fmt.Errorf("updating repeated command: %w", err)
		}
	} else {
		repeat = &models.RepeatedCommand{
			ChannelID:     s.Channel.ID,
			CommandInfoID: info.ID,
			Enabled:       true,
			Delay:         delay,
			MessageDiff:   messageDiff,
			LastCount:     s.Channel.MessageCount,
			Creator:       s.User,
			Editor:        s.User,
		}

		if err := repeat.Insert(ctx, s.Tx, boil.Infer()); err != nil {
			return fmt.Errorf("inserting repeated command: %w", err)
		}
	}

	if err := s.Deps.AddRepeat(ctx, repeat.ID, repeat.UpdatedAt, time.Duration(delay)*time.Second); err != nil {
		return err
	}

	dUnit := "message has passed."
	if messageDiff != 1 {
		dUnit = "messages have passed."
	}

	return s.Replyf(ctx, "Command '%s' will now repeat every %d seconds if at least %d %s", name, delay, messageDiff, dUnit)
}

func cmdRepeatDelete(ctx context.Context, s *session, cmd string, args string) error {
	name, _ := splitSpace(args)
	name = cleanCommandName(name)

	if name == "" {
		return s.ReplyUsage(ctx, "<name>")
	}

	info, repeat, err := findRepeatedCommand(ctx, name, s)
	if err != nil {
		return err
	}

	if info == nil {
		return s.Replyf(ctx, "Command '%s' does not exist.", name)
	}

	if repeat == nil {
		return s.Replyf(ctx, "Command '%s' has no repeat.", name)
	}

	if !s.UserLevel.CanAccessPG(info.AccessLevel) {
		al := flect.Pluralize(info.AccessLevel)
		return s.Replyf(ctx, "Command '%s' is restricted to %s; only %s and above can modify its repeat.", name, al, al)
	}

	if err := repeat.Delete(ctx, s.Tx); err != nil {
		return fmt.Errorf("deleting repeated command: %w", err)
	}

	if err := s.Deps.RemoveRepeat(ctx, repeat.ID); err != nil {
		return err
	}

	return s.Replyf(ctx, "Command '%s' will no longer repeat.", name)
}

func cmdRepeatOnOff(ctx context.Context, s *session, cmd string, args string) error {
	name, _ := splitSpace(args)
	name = cleanCommandName(name)

	if name == "" {
		return s.ReplyUsage(ctx, "<name>")
	}

	enable := cmd == "on"

	info, repeat, err := findRepeatedCommand(ctx, name, s)
	if err != nil {
		return err
	}

	if info == nil {
		return s.Replyf(ctx, "Command '%s' does not exist.", name)
	}

	if repeat == nil {
		return s.Replyf(ctx, "Command '%s' has no repeat.", name)
	}

	if !s.UserLevel.CanAccessPG(info.AccessLevel) {
		al := flect.Pluralize(info.AccessLevel)
		return s.Replyf(ctx, "Command '%s' is restricted to %s; only %s and above can modify its repeat.", name, al, al)
	}

	if repeat.Enabled == enable {
		if enable {
			return s.Replyf(ctx, "Repeated command '%s' is already enabled.", name)
		}
		return s.Replyf(ctx, "Repeated command '%s' is already disabled.", name)
	}

	repeat.Enabled = enable
	repeat.LastCount = s.Channel.MessageCount
	repeat.Editor = s.User
	repeat.InitTimestamp = null.Time{}

	columns := boil.Whitelist(
		models.RepeatedCommandColumns.UpdatedAt,
		models.RepeatedCommandColumns.Enabled,
		models.RepeatedCommandColumns.LastCount,
		models.RepeatedCommandColumns.Editor,
		models.RepeatedCommandColumns.InitTimestamp,
	)

	if err := repeat.Update(ctx, s.Tx, columns); err != nil {
		return fmt.Errorf("updating repeated command: %w", err)
	}

	if enable {
		err = s.Deps.AddRepeat(ctx, repeat.ID, repeat.UpdatedAt, time.Duration(repeat.Delay)*time.Second)
	} else {
		err = s.Deps.RemoveRepeat(ctx, repeat.ID)
	}

	if err != nil {
		return err
	}

	if enable {
		return s.Replyf(ctx, "Repeated command '%s' is now enabled.", name)
	}

	return s.Replyf(ctx, "Repeated command '%s' is now disabled.", name)
}

func cmdRepeatList(ctx context.Context, s *session, cmd string, args string) error {
	repeats, err := s.Channel.RepeatedCommands(
		qm.Load(models.RepeatedCommandRels.CommandInfo),
	).All(ctx, s.Tx)
	if err != nil {
		return fmt.Errorf("getting repeated commands: %w", err)
	}

	if len(repeats) == 0 {
		return s.Reply(ctx, "There are no repeated commands.")
	}

	slices.SortFunc(repeats, func(a, b *models.RepeatedCommand) int {
		return strings.Compare(a.R.CommandInfo.Name, b.R.CommandInfo.Name)
	})

	var builder strings.Builder
	builder.WriteString("Repeated commands: ")

	for i, repeat := range repeats {
		if i != 0 {
			builder.WriteString(", ")
		}

		builder.WriteString(repeat.R.CommandInfo.Name)
		builder.WriteString(" [")

		if repeat.Enabled {
			builder.WriteString("ON")
		} else {
			builder.WriteString("OFF")
		}

		builder.WriteString("] (")
		builder.WriteString(strconv.Itoa(repeat.Delay))
		builder.WriteByte(')')
	}

	return s.Reply(ctx, builder.String())
}

func findRepeatedCommand(ctx context.Context, name string, s *session) (*models.CommandInfo, *models.RepeatedCommand, error) {
	info, err := s.Channel.CommandInfos(
		models.CommandInfoWhere.Name.EQ(name),
		qm.Load(models.CommandInfoRels.RepeatedCommand),
		qm.For("UPDATE"),
	).One(ctx, s.Tx)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil, nil
	}

	if err != nil {
		return nil, nil, fmt.Errorf("getting command info: %w", err)
	}

	return info, info.R.RepeatedCommand, nil
}
