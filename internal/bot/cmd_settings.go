package bot

import (
	"context"
	"strconv"
	"strings"

	"github.com/gobuffalo/flect"
	"github.com/hortbot/hortbot/internal/db/models"
	"github.com/volatiletech/null"
	"github.com/volatiletech/sqlboiler/boil"
)

var settingCommands = newHandlerMap(map[string]handlerFunc{
	"filter":             {fn: cmdFilter, minLevel: levelModerator},
	"prefix":             {fn: cmdSettingPrefix, minLevel: levelBroadcaster},
	"bullet":             {fn: cmdSettingBullet, minLevel: levelBroadcaster},
	"cooldown":           {fn: cmdSettingCooldown, minLevel: levelModerator},
	"shouldmoderate":     {fn: cmdSettingShouldModerate, minLevel: levelModerator},
	"lastfm":             {fn: cmdSettingLastFM, minLevel: levelModerator},
	"parseyoutube":       {fn: cmdSettingParseYoutube, minLevel: levelModerator},
	"enablewarnings":     {fn: cmdSettingEnableWarnings, minLevel: levelModerator},
	"displaywarnings":    {fn: cmdSettingDisplayWarnings, minLevel: levelModerator},
	"timeoutduration":    {fn: cmdSettingTimeoutDuration, minLevel: levelModerator},
	"extralifeid":        {fn: cmdSettingExtraLifeID, minLevel: levelModerator},
	"subsmaylink":        {fn: cmdSettingSubsMayLink, minLevel: levelModerator},
	"subsregsminuslinks": {fn: cmdSettingSubsMayLink, minLevel: levelModerator},
	"mode":               {fn: cmdSettingMode, minLevel: levelModerator},
	"roll":               {fn: cmdSettingsRoll, minLevel: levelModerator},
	"steam":              {fn: cmdSettingsSteam, minLevel: levelModerator},
	"urban":              {fn: cmdSettingUrban, minLevel: levelModerator},
})

func cmdSettings(ctx context.Context, s *session, cmd string, args string) error {
	subcommand, args := splitSpace(args)
	subcommand = strings.ToLower(subcommand)

	if subcommand == "" {
		return s.ReplyUsage("<setting> <value>")
	}

	ok, err := settingCommands.Run(ctx, s, subcommand, args)
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

func cmdSettingTimeoutDuration(ctx context.Context, s *session, cmd string, args string) error {
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

func cmdSettingExtraLifeID(ctx context.Context, s *session, cmd string, args string) error {
	if args == "" {
		if s.Channel.ExtraLifeID == 0 {
			return s.Reply("Extra Life ID is not set.")
		}
		return s.Replyf("Extra Life ID is set to %d.", s.Channel.ExtraLifeID)
	}

	id, err := strconv.Atoi(args)
	if err != nil || id < 0 {
		return s.ReplyUsage("<participant ID>")
	}

	s.Channel.ExtraLifeID = id

	if err := s.Channel.Update(ctx, s.Tx, boil.Whitelist(models.ChannelColumns.UpdatedAt, models.ChannelColumns.ExtraLifeID)); err != nil {
		return err
	}

	if id == 0 {
		return s.Reply("Extra Life ID has been unset.")
	}

	return s.Replyf("Extra Life ID changed to %d.", id)
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
		return s.Replyf("Mode is set to %s.", s.Channel.Mode)
	}

	var newMode accessLevel

	switch args {
	case "0", "owner", "broadcaster":
		newMode = levelBroadcaster
	case "1", "mod", "mods", "moderators":
		newMode = levelModerator
	case "2", "everyone", "all":
		newMode = levelEveryone
	case "3", "subs", "subscribers", "regs", "regulars":
		newMode = levelSubscriber
	default:
		return s.Replyf("%s is not a valid mode.", args)
	}

	newModePG := newMode.PGEnum()

	if s.Channel.Mode == newModePG {
		return s.Replyf("Mode is already set to %s.", newMode.PGEnum())
	}

	s.Channel.Mode = newModePG

	if err := s.Channel.Update(ctx, s.Tx, boil.Whitelist(models.ChannelColumns.UpdatedAt, models.ChannelColumns.Mode)); err != nil {
		return err
	}

	return s.Replyf("Mode set to %s.", newModePG)
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
		return s.ReplyUsage("on|off")
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

func cmdSettingsRoll(ctx context.Context, s *session, cmd string, args string) error {
	opt, args := splitSpace(args)

	var column string
	var reply string

	switch opt {
	case "default":
		if args == "" {
			return s.Replyf("Default roll size is set to %d.", s.Channel.RollDefault)
		}

		def, err := strconv.Atoi(args)
		if err != nil || def <= 0 {
			return s.ReplyUsage("default <num>")
		}

		s.Channel.RollDefault = def
		column = models.ChannelColumns.RollDefault
		reply = "Default roll size set to " + strconv.Itoa(def) + "."

	case "cooldown":
		if args == "" {
			return s.Replyf("Roll command cooldown is set to %d seconds.", s.Channel.RollCooldown)
		}

		cooldown, err := strconv.Atoi(args)
		if err != nil || cooldown < 0 {
			return s.ReplyUsage("cooldown <seconds>")
		}

		s.Channel.RollCooldown = cooldown
		column = models.ChannelColumns.RollCooldown
		reply = "Roll command cooldown set to " + strconv.Itoa(cooldown) + " seconds."

	case "userlevel":
		if args == "" {
			return s.Replyf("Roll command is available to %s and above.", flect.Pluralize(s.Channel.RollLevel))
		}

		args = strings.ToLower(args)
		level := parseLevelPG(args)
		if level == "" {
			return s.ReplyUsage("userlevel everyone|regulars|subs|mods|broadcaster|admin")
		}

		s.Channel.RollLevel = level
		column = models.ChannelColumns.RollLevel
		reply = "Roll command is now available to " + flect.Pluralize(level) + " and above."

	default:
		return s.ReplyUsage("default|cooldown|userlevel ...")
	}

	if err := s.Channel.Update(ctx, s.Tx, boil.Whitelist(models.ChannelColumns.UpdatedAt, column)); err != nil {
		return err
	}

	return s.Reply(reply)
}

func cmdSettingsSteam(ctx context.Context, s *session, cmd string, args string) error {
	id, _ := splitSpace(args)

	if id == "" {
		if s.Channel.SteamID == "" {
			return s.Reply("Steam ID is not set.")
		}
		return s.Replyf("Steam ID is set to %s.", s.Channel.SteamID)
	}

	if strings.EqualFold(id, "off") {
		id = ""
	}

	s.Channel.SteamID = id

	if err := s.Channel.Update(ctx, s.Tx, boil.Whitelist(models.ChannelColumns.UpdatedAt, models.ChannelColumns.SteamID)); err != nil {
		return err
	}

	if id == "" {
		return s.Reply("Steam ID unset.")
	}

	return s.Replyf("Steam ID set to %s.", id)
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
