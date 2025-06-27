package bot

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/aarondl/null/v8"
	"github.com/aarondl/sqlboiler/v4/boil"
	"github.com/gobuffalo/flect"
	"github.com/hortbot/hortbot/internal/cbp"
	"github.com/hortbot/hortbot/internal/db/models"
	"github.com/hortbot/hortbot/internal/db/modelsx"
)

var settingCommands = newHandlerMap(map[string]handlerFunc{
	"filter":             {fn: cmdFilter, minLevel: AccessLevelModerator},
	"prefix":             {fn: cmdSettingPrefix, minLevel: AccessLevelBroadcaster},
	"bullet":             {fn: cmdSettingBullet, minLevel: AccessLevelBroadcaster},
	"cooldown":           {fn: cmdSettingCooldown, minLevel: AccessLevelModerator},
	"shouldmoderate":     {fn: cmdSettingShouldModerate, minLevel: AccessLevelModerator},
	"lastfm":             {fn: cmdSettingLastFM, minLevel: AccessLevelModerator},
	"parseyoutube":       {fn: cmdSettingParseYoutube, minLevel: AccessLevelModerator},
	"enablewarnings":     {fn: cmdSettingEnableWarnings, minLevel: AccessLevelModerator},
	"displaywarnings":    {fn: cmdSettingDisplayWarnings, minLevel: AccessLevelModerator},
	"timeoutduration":    {fn: cmdSettingTimeoutDuration, minLevel: AccessLevelModerator},
	"extralifeid":        {fn: cmdSettingExtraLifeID, minLevel: AccessLevelModerator},
	"subsmaylink":        {fn: cmdSettingSubsMayLink, minLevel: AccessLevelModerator},
	"subsregsminuslinks": {fn: cmdSettingSubsRegsMinusLinks, minLevel: AccessLevelModerator},
	"mode":               {fn: cmdSettingMode, minLevel: AccessLevelModerator},
	"roll":               {fn: cmdSettingsRoll, minLevel: AccessLevelModerator},
	"steam":              {fn: cmdSettingsSteam, minLevel: AccessLevelModerator},
	"urban":              {fn: cmdSettingUrban, minLevel: AccessLevelModerator},
	"tweet":              {fn: cmdSettingTweet, minLevel: AccessLevelModerator},
})

func cmdSettings(ctx context.Context, s *session, cmd string, args string) error {
	subcommand, args := splitSpace(args)
	subcommand = strings.ToLower(subcommand)

	if subcommand == "" {
		return s.ReplyUsage(ctx, "<setting> <value>")
	}

	ok, err := settingCommands.Run(ctx, s, subcommand, args)
	if !ok {
		return s.Replyf(ctx, "No such setting '%s'.", subcommand)
	}

	return err
}

func cmdSettingBullet(ctx context.Context, s *session, cmd string, args string) error {
	args = strings.TrimSpace(args)

	if args == "" {
		var bullet string
		if s.Channel.Bullet.Valid {
			bullet = s.Channel.Bullet.String
		} else {
			bullet = s.defaultBullet() + " (default)"
		}

		return s.Replyf(ctx, "Bullet is %s", bullet)
	}

	switch args[0] {
	case '/', '.':
		return s.Reply(ctx, "Bullet cannot be a command.")
	}

	reset := strings.EqualFold(args, "reset")

	if reset {
		s.Channel.Bullet = null.String{}
	} else {
		s.Channel.Bullet = null.StringFrom(args)
	}

	if err := s.Channel.Update(ctx, s.Tx, boil.Whitelist(models.ChannelColumns.UpdatedAt, models.ChannelColumns.Bullet)); err != nil {
		return fmt.Errorf("updating channel: %w", err)
	}

	if reset {
		return s.Reply(ctx, "Bullet reset to default.")
	}

	return s.Replyf(ctx, "Bullet changed to %s", args)
}

func cmdSettingPrefix(ctx context.Context, s *session, cmd string, args string) error {
	args = strings.TrimSpace(args)

	if args == "" {
		return s.Replyf(ctx, "Prefix is %s", s.Channel.Prefix)
	}

	switch args[0] {
	case '/', '.':
		return s.Replyf(ctx, "Prefix cannot begin with %c", args[0])
	}

	reset := strings.EqualFold(args, "reset")

	if reset {
		s.Channel.Prefix = modelsx.DefaultPrefix
	} else {
		if utf8.RuneCountInString(args) != 1 {
			return s.Reply(ctx, "Prefix may only be a single character.")
		}
		s.Channel.Prefix = args
	}

	if err := s.Channel.Update(ctx, s.Tx, boil.Whitelist(models.ChannelColumns.UpdatedAt, models.ChannelColumns.Prefix)); err != nil {
		return fmt.Errorf("updating channel: %w", err)
	}

	if reset {
		return s.Replyf(ctx, "Prefix reset to %s", s.Channel.Prefix)
	}

	return s.Replyf(ctx, "Prefix changed to %s", args)
}

func cmdSettingCooldown(ctx context.Context, s *session, cmd string, args string) error {
	var cooldown null.Int

	if args == "" {
		cooldown = s.Channel.Cooldown
		if cooldown.Valid {
			return s.Replyf(ctx, "Cooldown is %d seconds.", s.Channel.Cooldown.Int)
		}
		return s.Replyf(ctx, "Cooldown is %d seconds (default).", s.Deps.DefaultCooldown)
	}

	reset := strings.EqualFold(args, "reset")

	if !reset {
		v, err := strconv.Atoi(args)
		if err != nil {
			return s.Reply(ctx, "New cooldown must be an integer.")
		}
		cooldown = null.IntFrom(v)
	}

	s.Channel.Cooldown = cooldown

	if err := s.Channel.Update(ctx, s.Tx, boil.Whitelist(models.ChannelColumns.UpdatedAt, models.ChannelColumns.Cooldown)); err != nil {
		return fmt.Errorf("updating channel: %w", err)
	}

	if reset {
		return s.Replyf(ctx, "Cooldown reset to %d seconds (default).", cooldown.Int)
	}

	return s.Replyf(ctx, "Cooldown changed to %d seconds.", cooldown.Int)
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
			return s.Reply(ctx, "LastFM user is not set.")
		}

		return s.Replyf(ctx, "LastFM user is set to %s.", lfm)

	case "off":
		s.Channel.LastFM = ""

	default:
		s.Channel.LastFM = args
	}

	if err := s.Channel.Update(ctx, s.Tx, boil.Whitelist(models.ChannelColumns.UpdatedAt, models.ChannelColumns.LastFM)); err != nil {
		return fmt.Errorf("updating channel: %w", err)
	}

	if args == "off" {
		return s.Reply(ctx, "LastFM support has been disabled.")
	}

	return s.Replyf(ctx, "LastFM user changed to %s.", args)
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

func cmdSettingTimeoutDuration(ctx context.Context, s *session, cmd string, args string) error {
	if args == "" {
		if s.Channel.TimeoutDuration == 0 {
			return s.Reply(ctx, "Timeout duration is set to Twitch default.")
		}
		return s.Replyf(ctx, "Timeout duration is set to %d seconds.", s.Channel.TimeoutDuration)
	}

	dur, err := strconv.Atoi(args)
	if err != nil {
		return s.ReplyUsage(ctx, "<seconds>")
	}

	if dur < 0 {
		return s.Reply(ctx, "Timeout duration must not be negative.")
	}

	s.Channel.TimeoutDuration = dur

	if err := s.Channel.Update(ctx, s.Tx, boil.Whitelist(models.ChannelColumns.UpdatedAt, models.ChannelColumns.TimeoutDuration)); err != nil {
		return fmt.Errorf("updating channel: %w", err)
	}

	if dur == 0 {
		return s.Reply(ctx, "Timeout duration changed to Twitch default.")
	}

	return s.Replyf(ctx, "Timeout duration changed to %d seconds.", dur)
}

func cmdSettingExtraLifeID(ctx context.Context, s *session, cmd string, args string) error {
	if args == "" {
		if s.Channel.ExtraLifeID == 0 {
			return s.Reply(ctx, "Extra Life ID is not set.")
		}
		return s.Replyf(ctx, "Extra Life ID is set to %d.", s.Channel.ExtraLifeID)
	}

	id, err := strconv.ParseInt(args, 10, 32)
	if err != nil || id < 0 {
		return s.ReplyUsage(ctx, "<participant ID>")
	}

	s.Channel.ExtraLifeID = int(id)

	if err := s.Channel.Update(ctx, s.Tx, boil.Whitelist(models.ChannelColumns.UpdatedAt, models.ChannelColumns.ExtraLifeID)); err != nil {
		return fmt.Errorf("updating channel: %w", err)
	}

	if id == 0 {
		return s.Reply(ctx, "Extra Life ID has been unset.")
	}

	return s.Replyf(ctx, "Extra Life ID changed to %d.", id)
}

func cmdSettingSubsRegsMinusLinks(ctx context.Context, s *session, cmd string, args string) error {
	return s.Reply(ctx, "This option has been removed; use subsMayLink instead.")
}

func cmdSettingSubsMayLink(ctx context.Context, s *session, cmd string, args string) error {
	return updateBoolean(
		ctx, s, cmd, args,
		func() bool { return s.Channel.SubsMayLink },
		func(v bool) { s.Channel.SubsMayLink = v },
		models.ChannelColumns.SubsMayLink,
		"subsMayLink",
		"Subs already may post links.",
		"Subs already may not post links.",
		"Subs may now post links.",
		"Subs may no longer post links.",
	)
}

func cmdSettingMode(ctx context.Context, s *session, cmd string, args string) error {
	if args == "" {
		return s.Replyf(ctx, "Mode is set to %s.", s.Channel.Mode)
	}

	var newMode AccessLevel

	switch args {
	case "0", "owner", "broadcaster":
		newMode = AccessLevelBroadcaster
	case "1", "mod", "mods", "moderators":
		newMode = AccessLevelModerator
	case "2", "everyone", "all":
		newMode = AccessLevelEveryone
	case "3", "subs", "subscribers", "regs", "regulars":
		newMode = AccessLevelSubscriber
	default:
		return s.Replyf(ctx, "%s is not a valid mode.", args)
	}

	newModePG := newMode.PGEnum()

	if s.Channel.Mode == newModePG {
		return s.Replyf(ctx, "Mode is already set to %s.", newMode.PGEnum())
	}

	s.Channel.Mode = newModePG

	if err := s.Channel.Update(ctx, s.Tx, boil.Whitelist(models.ChannelColumns.UpdatedAt, models.ChannelColumns.Mode)); err != nil {
		return fmt.Errorf("updating channel: %w", err)
	}

	return s.Replyf(ctx, "Mode set to %s.", newModePG)
}

func updateBoolean(
	ctx context.Context, s *session, _ string, args string,
	get func() bool, set func(v bool), column string,
	name, alreadyTrue, alreadyFalse, setTrue, setFalse string,
) error {
	args = strings.ToLower(args)

	v := false

	switch args {
	case "":
		return s.Replyf(ctx, "%s is set to %v.", name, get())

	case "on", "enabled", "true", "1", "yes":
		if get() {
			return s.Reply(ctx, alreadyTrue)
		}

		v = true

	case "off", "disabled", "false", "0", "no":
		if !get() {
			return s.Reply(ctx, alreadyFalse)
		}

	default:
		return s.ReplyUsage(ctx, "on|off")
	}

	set(v)

	if err := s.Channel.Update(ctx, s.Tx, boil.Whitelist(models.ChannelColumns.UpdatedAt, column)); err != nil {
		return fmt.Errorf("updating channel: %w", err)
	}

	if v {
		return s.Reply(ctx, setTrue)
	}

	return s.Reply(ctx, setFalse)
}

func cmdSettingsRoll(ctx context.Context, s *session, cmd string, args string) error {
	opt, args := splitSpace(args)

	var column string
	var reply string

	switch opt {
	case "default":
		if args == "" {
			return s.Replyf(ctx, "Default roll size is set to %d.", s.Channel.RollDefault)
		}

		def, err := strconv.Atoi(args)
		if err != nil || def <= 0 {
			return s.ReplyUsage(ctx, "default <num>")
		}

		s.Channel.RollDefault = def
		column = models.ChannelColumns.RollDefault
		reply = "Default roll size set to " + strconv.Itoa(def) + "."

	case "cooldown":
		if args == "" {
			return s.Replyf(ctx, "Roll command cooldown is set to %d seconds.", s.Channel.RollCooldown)
		}

		cooldown, err := strconv.Atoi(args)
		if err != nil || cooldown < 0 {
			return s.ReplyUsage(ctx, "cooldown <seconds>")
		}

		s.Channel.RollCooldown = cooldown
		column = models.ChannelColumns.RollCooldown
		reply = "Roll command cooldown set to " + strconv.Itoa(cooldown) + " seconds."

	case "userlevel":
		if args == "" {
			return s.Replyf(ctx, "Roll command is available to %s and above.", flect.Pluralize(s.Channel.RollLevel))
		}

		args = strings.ToLower(args)
		level := parseLevelPG(args)
		if level == "" {
			return s.ReplyUsage(ctx, "userlevel everyone|regulars|subs|vips|mods|broadcaster|admin")
		}

		s.Channel.RollLevel = level
		column = models.ChannelColumns.RollLevel
		reply = "Roll command is now available to " + flect.Pluralize(level) + " and above."

	default:
		return s.ReplyUsage(ctx, "default|cooldown|userlevel ...")
	}

	if err := s.Channel.Update(ctx, s.Tx, boil.Whitelist(models.ChannelColumns.UpdatedAt, column)); err != nil {
		return fmt.Errorf("updating channel: %w", err)
	}

	return s.Reply(ctx, reply)
}

func cmdSettingsSteam(ctx context.Context, s *session, cmd string, args string) error {
	id, _ := splitSpace(args)

	if id == "" {
		if s.Channel.SteamID == "" {
			return s.Reply(ctx, "Steam ID is not set.")
		}
		return s.Replyf(ctx, "Steam ID is set to %s.", s.Channel.SteamID)
	}

	if strings.EqualFold(id, "off") {
		id = ""
	}

	s.Channel.SteamID = id

	if err := s.Channel.Update(ctx, s.Tx, boil.Whitelist(models.ChannelColumns.UpdatedAt, models.ChannelColumns.SteamID)); err != nil {
		return fmt.Errorf("updating channel: %w", err)
	}

	if id == "" {
		return s.Reply(ctx, "Steam ID unset.")
	}

	return s.Replyf(ctx, "Steam ID set to %s.", id)
}

func cmdSettingUrban(ctx context.Context, s *session, cmd string, args string) error {
	return updateBoolean(
		ctx, s, cmd, args,
		func() bool { return s.Channel.UrbanEnabled },
		func(v bool) { s.Channel.UrbanEnabled = v },
		models.ChannelColumns.UrbanEnabled,
		"urban",
		"Urban Dictionary is already enabled.",
		"Urban Dictionary is already disabled.",
		"Urban Dictionary is now enabled.",
		"Urban Dictionary is now disabled.",
	)
}

func cmdSettingTweet(ctx context.Context, s *session, cmd string, args string) error {
	if args == "" {
		return s.Replyf(ctx, "Tweet is set to: %s", s.Channel.Tweet)
	}

	var warning string
	if _, malformed := cbp.Parse(args); malformed {
		warning += " - Warning: message contains stray (_ or _) separators and may not be processed correctly."
	}

	s.Channel.Tweet = args

	if err := s.Channel.Update(ctx, s.Tx, boil.Whitelist(models.ChannelColumns.UpdatedAt, models.ChannelColumns.Tweet)); err != nil {
		return fmt.Errorf("updating channel: %w", err)
	}

	return s.Replyf(ctx, `Tweet set to: "%s"%s`, args, warning)
}
