package bot

import (
	"context"
	"database/sql"
	"strconv"
	"strings"

	"github.com/hortbot/hortbot/internal/db/models"
	"github.com/hortbot/hortbot/internal/version"
	"github.com/volatiletech/sqlboiler/boil"
)

var adminCommands handlerMap

func init() {
	// To prevent initialization loop.
	adminCommands = newHandlerMap(map[string]handlerFunc{
		"roundtrip":     {fn: cmdAdminRoundtrip, minLevel: levelAdmin},
		"block":         {fn: cmdAdminBlock, minLevel: levelAdmin},
		"unblock":       {fn: cmdAdminUnblock, minLevel: levelAdmin},
		"channels":      {fn: cmdAdminChannels, minLevel: levelAdmin},
		"color":         {fn: cmdAdminColor, minLevel: levelAdmin},
		"spam":          {fn: cmdAdminSpam, minLevel: levelAdmin},
		"imp":           {fn: cmdAdminImp, minLevel: levelAdmin},
		"publicjoin":    {fn: cmdAdminPublicJoin, minLevel: levelAdmin},
		"version":       {fn: cmdAdminVersion, minLevel: levelAdmin},
		"reloadrepeats": {fn: cmdAdminReloadRepeats, minLevel: levelAdmin},
	})
}

func cmdAdmin(ctx context.Context, s *session, cmd string, args string) error {
	subcommand, args := splitSpace(args)
	subcommand = strings.ToLower(subcommand)

	ok, err := adminCommands.Run(ctx, s, subcommand, args)
	if ok || err != nil {
		return err
	}

	if strings.HasPrefix(subcommand, "#") {
		return s.Deps.Sender.SendMessage(ctx, s.Origin, subcommand, args)
	}

	return s.Replyf(ctx, "Bad command %s", subcommand)
}

func cmdAdminRoundtrip(ctx context.Context, s *session, cmd string, args string) error {
	now := s.Deps.Clock.Now()
	return s.Replyf(ctx, "total=%v, handle=%v", now.Sub(s.TMISent), now.Sub(s.Start))
}

func cmdAdminBlock(ctx context.Context, s *session, cmd string, args string) error {
	if args == "" {
		return s.ReplyUsage(ctx, "<username>")
	}

	u, err := s.Deps.Twitch.GetUserForUsername(ctx, args)
	if err != nil {
		return s.Replyf(ctx, "Error getting ID from Twitch: %s", err.Error())
	}

	bu := &models.BlockedUser{TwitchID: u.ID}
	if err := bu.Upsert(ctx, s.Tx, false, []string{models.BlockedUserColumns.TwitchID}, boil.Infer(), boil.Infer()); err != nil {
		return err
	}

	channel, err := models.Channels(models.ChannelWhere.UserID.EQ(u.ID)).One(ctx, s.Tx)
	if err != nil && err != sql.ErrNoRows {
		return err
	}

	if err != sql.ErrNoRows && channel.Active {
		channel.Active = false

		if err := channel.Update(ctx, s.Tx, boil.Whitelist(models.ChannelColumns.UpdatedAt, models.ChannelColumns.Active)); err != nil {
			return err
		}

		if err := s.Deps.Notifier.NotifyChannelUpdates(ctx, channel.BotName); err != nil {
			return err
		}
	}

	return s.Replyf(ctx, "%s (%d) has been blocked.", u.DispName(), u.ID)
}

func cmdAdminUnblock(ctx context.Context, s *session, cmd string, args string) error {
	if args == "" {
		return s.ReplyUsage(ctx, "<username>")
	}

	u, err := s.Deps.Twitch.GetUserForUsername(ctx, args)
	if err != nil {
		return s.Replyf(ctx, "Error getting ID from Twitch: %s", err.Error())
	}

	if err := models.BlockedUsers(models.BlockedUserWhere.TwitchID.EQ(u.ID)).DeleteAll(ctx, s.Tx); err != nil {
		return err
	}

	return s.Replyf(ctx, "%s (%d) has been unblocked.", u.DispName(), u.ID)
}

func cmdAdminChannels(ctx context.Context, s *session, cmd string, args string) error {
	count, err := models.Channels(models.ChannelWhere.Active.EQ(true)).Count(ctx, s.Tx)
	if err != nil {
		return err
	}

	ch := "channels"
	if count == 1 {
		ch = "channel"
	}

	return s.Replyf(ctx, "Currently in %d %s.", count, ch)
}

func cmdAdminColor(ctx context.Context, s *session, cmd string, args string) error {
	if args == "" {
		return s.ReplyUsage(ctx, "<color>")
	}

	if err := s.SendCommand(ctx, "color", args); err != nil {
		return err
	}

	return s.Replyf(ctx, "Color set to %s.", args)
}

func cmdAdminSpam(ctx context.Context, s *session, cmd string, args string) error {
	usage := func() error {
		return s.ReplyUsage(ctx, "<num> <message>")
	}

	if args == "" {
		return usage()
	}

	numStr, message := splitSpace(args)

	if numStr == "" || message == "" {
		return usage()
	}

	n, err := strconv.Atoi(numStr)
	if err != nil || n <= 0 {
		return usage()
	}

	var builder strings.Builder

	for i := 0; i < n; i++ {
		if i != 0 {
			builder.WriteByte(' ')
		}
		builder.WriteString(message)
	}

	return s.Reply(ctx, builder.String())
}

func cmdAdminImp(ctx context.Context, s *session, cmd string, args string) error {
	name, msg := splitSpace(args)
	name = strings.ToLower(name)

	if name == "" {
		return s.ReplyUsage(ctx, "<channel> <message>")
	}

	otherChannel, err := models.Channels(models.ChannelWhere.Name.EQ(name)).One(ctx, s.Tx)
	if err != nil {
		if err == sql.ErrNoRows {
			return s.Replyf(ctx, "Channel %s does not exist.", name)
		}
		return err
	}

	s.RoomID = otherChannel.UserID
	s.RoomIDStr = strconv.FormatInt(s.RoomID, 10)
	s.Message = msg
	s.Imp = true

	return handleSession(ctx, s)
}

func cmdAdminPublicJoin(ctx context.Context, s *session, cmd string, args string) error {
	onOff, args := splitSpace(args)
	botName, _ := splitSpace(args)

	onOff = strings.ToLower(onOff)
	botName = strings.ToLower(botName)

	switch botName {
	case "":
		botName = s.Origin
	case "default":
		botName = ""
	}

	unset := false
	enable := false

	switch onOff {
	case "on":
		enable = true
	case "off":
	case "unset":
		unset = true
	default:
		v, err := s.Deps.Redis.PublicJoin(ctx, botName)
		if err != nil {
			return err
		}

		if v == nil {
			return s.Reply(ctx, "Public join unset.")
		}

		return s.Replyf(ctx, "Public join set to: %v", *v)
	}

	action := "unset"

	if unset {
		if err := s.Deps.Redis.UnsetPublicJoin(ctx, botName); err != nil {
			return err
		}
	} else {
		if enable {
			action = "enabled"
		} else {
			action = "disabled"
		}

		if err := s.Deps.Redis.SetPublicJoin(ctx, botName, enable); err != nil {
			return err
		}
	}

	if botName != "" {
		return s.Replyf(ctx, "Public join %s for %s.", action, botName)
	}

	return s.Replyf(ctx, "Default public join %s.", action)
}

func cmdAdminVersion(ctx context.Context, s *session, _ string, _ string) error {
	return s.Replyf(ctx, "hortbot version %s", version.Version())
}

func cmdAdminReloadRepeats(ctx context.Context, s *session, _ string, _ string) error {
	before := s.Deps.Clock.Now()

	if err := s.Deps.ReloadRepeats(ctx); err != nil {
		return s.Replyf(ctx, "Error reloading repeats: %s", err.Error())
	}

	repeats, schedules := s.Deps.CountRepeats()
	return s.Replyf(ctx, "Reloaded %d repeats and %d schedules in %v.", repeats, schedules, s.Deps.Clock.Since(before))
}
