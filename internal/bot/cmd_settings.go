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
	"prefix":          {fn: cmdSettingPrefix, minLevel: levelBroadcaster},
	"bullet":          {fn: cmdSettingBullet, minLevel: levelBroadcaster},
	"cooldown":        {fn: cmdSettingCooldown, minLevel: levelModerator},
	"shouldmoderate":  {fn: cmdSettingShouldModerate, minLevel: levelModerator},
	"lastfm":          {fn: cmdSettingLastFM, minLevel: levelModerator},
	"parseyoutube":    {fn: cmdSettingParseYoutube, minLevel: levelModerator},
	"enablewarnings":  {fn: cmdSettingEnableWarnings, minLevel: levelModerator},
	"displaywarnings": {fn: cmdSettingDisplayWarnings, minLevel: levelModerator},
	"timeoutduration": {fn: cmdSettingsTimeoutDuration, minLevel: levelModerator},
	"filter":          {fn: cmdFilter, minLevel: levelModerator},
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
	return updateBoolean(
		ctx, s, cmd, args,
		func() bool { return s.Channel.ShouldModerate },
		func(v bool) { s.Channel.ShouldModerate = v },
		models.ChannelColumns.ShouldModerate,
		"shouldModerate",
		s.Channel.BotName+" is already moderating.",
		s.Channel.BotName+" is already not moderating.",
		s.Channel.BotName+" will attempt to moderate in this channel.",
		s.Channel.BotName+" will no longer attempt to moderate in this channel.",
	)
}

func cmdSettingLastFM(ctx context.Context, s *session, cmd string, args string) error {
	args = strings.ToLower(args)

	switch args {
	case "":
		lfm := s.Channel.LastFM

		if lfm == "" {
			return s.Reply("LastFM user is not set.")
		}

		return s.Replyf("LastFM user is set to %s.", lfm)

	case "off":
		s.Channel.LastFM = ""

	default:
		s.Channel.LastFM = args
	}

	if err := s.Channel.Update(ctx, s.Tx, boil.Whitelist(models.ChannelColumns.UpdatedAt, models.ChannelColumns.LastFM)); err != nil {
		return err
	}

	if args == "off" {
		return s.Reply("LastFM support has been disabled.")
	}

	return s.Replyf("LastFM user changed to %s.", args)
}

func cmdSettingParseYoutube(ctx context.Context, s *session, cmd string, args string) error {
	return updateBoolean(
		ctx, s, cmd, args,
		func() bool { return s.Channel.ParseYoutube },
		func(v bool) { s.Channel.ParseYoutube = v },
		models.ChannelColumns.ParseYoutube,
		"parseYoutube",
		"YouTube link parsing is already enabled.",
		"YouTube link parsing is already disabled.",
		"YouTube link parsing is now enabled.",
		"YouTube link parsing is now disabled.",
	)
}

func cmdSettingEnableWarnings(ctx context.Context, s *session, cmd string, args string) error {
	return updateBoolean(
		ctx, s, cmd, args,
		func() bool { return s.Channel.EnableWarnings },
		func(v bool) { s.Channel.EnableWarnings = v },
		models.ChannelColumns.EnableWarnings,
		"enableWarnings",
		"Warnings are already enabled.",
		"Warnings are already disabled.",
		"Warnings are now enabled.",
		"Warnings are now disabled.",
	)
}

func cmdSettingDisplayWarnings(ctx context.Context, s *session, cmd string, args string) error {
	return updateBoolean(
		ctx, s, cmd, args,
		func() bool { return s.Channel.DisplayWarnings },
		func(v bool) { s.Channel.DisplayWarnings = v },
		models.ChannelColumns.DisplayWarnings,
		"displayWarnings",
		"Warning/timeout messages are already enabled.",
		"Warning/timeout messages are already disabled.",
		"Warning/timeout messages are now enabled.",
		"Warning/timeout messages are now disabled.",
	)
}

func cmdSettingsTimeoutDuration(ctx context.Context, s *session, cmd string, args string) error {
	if args == "" {
		if s.Channel.TimeoutDuration == 0 {
			return s.Reply("Timeout duration is set to Twitch default.")
		}
		return s.Replyf("Timeout duration is set to %d seconds.", s.Channel.TimeoutDuration)
	}

	dur, err := strconv.Atoi(args)
	if err != nil {
		return s.ReplyUsage("<seconds>")
	}

	if dur < 0 {
		return s.Reply("Timeout duration must not be negative.")
	}

	s.Channel.TimeoutDuration = dur

	if err := s.Channel.Update(ctx, s.Tx, boil.Whitelist(models.ChannelColumns.UpdatedAt, models.ChannelColumns.TimeoutDuration)); err != nil {
		return err
	}

	if dur == 0 {
		return s.Reply("Timeout duration changed to Twitch default.")
	}

	return s.Replyf("Timeout duration changed to %d seconds.", dur)
}

func updateBoolean(
	ctx context.Context, s *session, cmd string, args string,
	get func() bool, set func(v bool), column string,
	name, alreadyTrue, alreadyFalse, setTrue, setFalse string,
) error {
	args = strings.ToLower(args)

	v := false

	switch args {
	case "":
		return s.Replyf("%s is set to %v.", name, get())

	case "on", "enabled", "true", "1", "yes":
		if get() {
			return s.Reply(alreadyTrue)
		}

		v = true

	case "off", "disabled", "false", "0", "no":
		if !get() {
			return s.Reply(alreadyFalse)
		}

	default:
		return s.ReplyUsage("<on|off>")
	}

	set(v)

	if err := s.Channel.Update(ctx, s.Tx, boil.Whitelist(models.ChannelColumns.UpdatedAt, column)); err != nil {
		return err
	}

	if v {
		return s.Reply(setTrue)
	}

	return s.Reply(setFalse)
}
