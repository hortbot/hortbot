package bot

import (
	"context"
	"strconv"
	"strings"

	"github.com/hortbot/hortbot/internal/db/models"
	"github.com/volatiletech/null"
	"github.com/volatiletech/sqlboiler/boil"
)

var builtinSettings builtinMap = map[string]builtinCommand{
	"prefix":   {fn: cmdPrefix, minLevel: LevelBroadcaster},
	"bullet":   {fn: cmdBullet, minLevel: LevelBroadcaster},
	"cooldown": {fn: cmdCooldown, minLevel: LevelModerator},
}

func cmdSettings(ctx context.Context, s *Session, cmd string, args string) error {
	subcommand, args := splitSpace(args)

	if subcommand == "" {
		return s.ReplyUsage(cmd + " <setting> <value>")
	}

	ok, err := builtinSettings.run(ctx, s, subcommand, args)
	if !ok {
		return s.Replyf("no such setting %s", subcommand)
	}

	return err
}

func cmdBullet(ctx context.Context, s *Session, cmd string, args string) error {
	args = strings.TrimSpace(args)

	if args == "" {
		var bullet string
		if s.Channel.Bullet.Valid {
			bullet = s.Channel.Bullet.String
		} else {
			bullet = s.Bot.bullet + " (default)"
		}

		return s.Replyf("bullet is %s", bullet)
	}

	switch args[0] {
	case '/', '.':
		return s.Reply("bullet cannot be a command")
	}

	reset := strings.EqualFold(args, "reset")

	if reset {
		s.Channel.Bullet = null.String{}
	} else {
		s.Channel.Bullet = null.StringFrom(args)
	}

	if err := s.Channel.Update(ctx, s.Tx, boil.Whitelist(models.ChannelColumns.UpdatedAt, models.ChannelColumns.Bullet)); err != nil {
		return err
	}

	if reset {
		return s.Reply("bullet reset to default")
	}

	return s.Replyf("bullet changed to %s", args)
}

func cmdPrefix(ctx context.Context, s *Session, cmd string, args string) error {
	args = strings.TrimSpace(args)

	if args == "" {
		return s.Replyf("prefix is %s", s.Channel.Prefix)
	}

	switch args[0] {
	case '/', '.':
		return s.Replyf("prefix cannot begin with %c", args[0])
	}

	reset := strings.EqualFold(args, "reset")

	if reset {
		s.Channel.Prefix = s.Bot.prefix
	} else {
		s.Channel.Prefix = args
	}

	if err := s.Channel.Update(ctx, s.Tx, boil.Whitelist(models.ChannelColumns.UpdatedAt, models.ChannelColumns.Prefix)); err != nil {
		return err
	}

	if reset {
		return s.Replyf("prefix reset to %s", s.Channel.Prefix)
	}

	return s.Replyf("prefix changed to %s", args)
}

func cmdCooldown(ctx context.Context, s *Session, cmd string, args string) error {
	args = strings.TrimSpace(args)

	var cooldown null.Int

	if args == "" {
		cooldown = s.Channel.Cooldown
		if cooldown.Valid {
			return s.Replyf("cooldown is %ds", s.Channel.Cooldown.Int)
		}
		return s.Replyf("cooldown is %ds (default)", s.Bot.cooldown)
	}

	reset := strings.EqualFold(args, "reset")

	if !reset {
		v, err := strconv.Atoi(args)
		if err != nil {
			return s.Reply("new cooldown must be an integer")
		}
		cooldown = null.IntFrom(v)
	}

	s.Channel.Cooldown = cooldown

	if err := s.Channel.Update(ctx, s.Tx, boil.Whitelist(models.ChannelColumns.UpdatedAt, models.ChannelColumns.Cooldown)); err != nil {
		return err
	}

	if reset {
		return s.Replyf("cooldown reset to %ds (default)", cooldown.Int)
	}

	return s.Replyf("cooldown changed to %ds", cooldown.Int)
}
