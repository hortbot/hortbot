package bot

import (
	"context"
	"strconv"
	"strings"

	"github.com/hortbot/hortbot/internal/db/models"
	"github.com/volatiletech/null"
	"github.com/volatiletech/sqlboiler/boil"
)

var settingCommands handlerMap = map[string]handlerFunc{
	"prefix":         {fn: cmdSettingPrefix, minLevel: levelBroadcaster},
	"bullet":         {fn: cmdSettingBullet, minLevel: levelBroadcaster},
	"cooldown":       {fn: cmdSettingCooldown, minLevel: levelModerator},
	"shouldmoderate": {fn: cmdSettingShouldModerate, minLevel: levelModerator},
}

func cmdSettings(ctx context.Context, s *session, cmd string, args string) error {
	subcommand, args := splitSpace(args)
	subcommand = strings.ToLower(subcommand)

	if subcommand == "" {
		return s.ReplyUsage("<setting> <value>")
	}

	ok, err := settingCommands.run(ctx, s, subcommand, args)
	if !ok {
		return s.Replyf("No such setting '%s'.", subcommand)
	}

	return err
}

func cmdSettingBullet(ctx context.Context, s *session, cmd string, args string) error {
	if args == "" {
		var bullet string
		if s.Channel.Bullet.Valid {
			bullet = s.Channel.Bullet.String
		} else {
			bullet = s.Deps.DefaultBullet + " (default)"
		}

		return s.Replyf("Bullet is %s", bullet)
	}

	switch args[0] {
	case '/', '.':
		return s.Reply("Bullet cannot be a command.")
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
		return s.Reply("Bullet reset to default.")
	}

	return s.Replyf("Bullet changed to %s", args)
}

func cmdSettingPrefix(ctx context.Context, s *session, cmd string, args string) error {
	if args == "" {
		return s.Replyf("Prefix is %s", s.Channel.Prefix)
	}

	switch args[0] {
	case '/', '.':
		return s.Replyf("Prefix cannot begin with %c", args[0])
	}

	reset := strings.EqualFold(args, "reset")

	if reset {
		s.Channel.Prefix = s.Deps.DefaultPrefix
	} else {
		s.Channel.Prefix = args
	}

	if err := s.Channel.Update(ctx, s.Tx, boil.Whitelist(models.ChannelColumns.UpdatedAt, models.ChannelColumns.Prefix)); err != nil {
		return err
	}

	if reset {
		return s.Replyf("Prefix reset to %s", s.Channel.Prefix)
	}

	return s.Replyf("Prefix changed to %s", args)
}

func cmdSettingCooldown(ctx context.Context, s *session, cmd string, args string) error {
	var cooldown null.Int

	if args == "" {
		cooldown = s.Channel.Cooldown
		if cooldown.Valid {
			return s.Replyf("Cooldown is %d seconds.", s.Channel.Cooldown.Int)
		}
		return s.Replyf("Cooldown is %d seconds (default).", s.Deps.DefaultCooldown)
	}

	reset := strings.EqualFold(args, "reset")

	if !reset {
		v, err := strconv.Atoi(args)
		if err != nil {
			return s.Reply("New cooldown must be an integer.")
		}
		cooldown = null.IntFrom(v)
	}

	s.Channel.Cooldown = cooldown

	if err := s.Channel.Update(ctx, s.Tx, boil.Whitelist(models.ChannelColumns.UpdatedAt, models.ChannelColumns.Cooldown)); err != nil {
		return err
	}

	if reset {
		return s.Replyf("Cooldown reset to %d seconds (default).", cooldown.Int)
	}

	return s.Replyf("Cooldown changed to %d seconds.", cooldown.Int)
}

func cmdSettingShouldModerate(ctx context.Context, s *session, cmd string, args string) error {
	args = strings.ToLower(args)

	switch args {
	case "":
		return s.Replyf("shouldModerate is set to %v.", s.Channel.ShouldModerate)

	case "on", "enabled", "true", "1", "yes":
		if s.Channel.ShouldModerate {
			return s.Replyf("%s is already moderating.", s.Channel.BotName)
		}
		s.Channel.ShouldModerate = true

	case "off", "disabled", "false", "0", "no":
		if !s.Channel.ShouldModerate {
			return s.Replyf("%s is already not moderating.", s.Channel.BotName)
		}
		s.Channel.ShouldModerate = false

	default:
		return s.ReplyUsage("<on|off>")
	}

	if err := s.Channel.Update(ctx, s.Tx, boil.Whitelist(models.ChannelColumns.UpdatedAt, models.ChannelColumns.ShouldModerate)); err != nil {
		return err
	}

	if s.Channel.ShouldModerate {
		return s.Replyf("%s will attempt to moderate in this channel.", s.Channel.BotName)
	}

	return s.Replyf("%s will no longer attempt to moderate in this channel.", s.Channel.BotName)
}
