package bot

import (
	"context"
	"database/sql"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/hako/durafmt"
	"github.com/hortbot/hortbot/internal/db/models"
	"github.com/hortbot/hortbot/internal/db/modelsx"
	"github.com/hortbot/hortbot/internal/version"
	"github.com/volatiletech/sqlboiler/boil"
	"github.com/volatiletech/sqlboiler/queries/qm"
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
		"version":       {fn: cmdAdminVersion, minLevel: levelAdmin},
		"reloadrepeats": {fn: cmdAdminReloadRepeats, minLevel: levelAdmin},
		"deletechannel": {fn: cmdAdminDeleteChannel, minLevel: levelAdmin},
		"sleep":         {fn: cmdAdminSleep, minLevel: levelAdmin},
		"syncjoined":    {fn: cmdAdminSyncJoined, minLevel: levelAdmin},
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
	s.sendRoundtrip = true
	return nil
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

	ch := pluralInt64(count, "channel", "channels")
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
	s.Message = msg
	s.Imp = true

	return handleSession(ctx, s)
}

func cmdAdminVersion(ctx context.Context, s *session, _ string, _ string) error {
	return s.Replyf(ctx, "hortbot version %s, built with %s", version.Version(), runtime.Version())
}

func cmdAdminReloadRepeats(ctx context.Context, s *session, _ string, _ string) error {
	before := s.Deps.Clock.Now()

	if err := s.Deps.ReloadRepeats(ctx); err != nil {
		return s.Replyf(ctx, "Error reloading repeats: %s", err.Error())
	}

	repeats, schedules, err := s.Deps.CountRepeats(ctx)
	if err != nil {
		return err
	}

	repeatStr := pluralInt(repeats, "repeat", "repeats")
	scheduleStr := pluralInt(schedules, "schedule", "schedules")

	return s.Replyf(ctx, "Reloaded %d %s and %d %s in %v.", repeats, repeatStr, schedules, scheduleStr, s.Deps.Clock.Since(before))
}

const deleteChannelConfirmDur = 10 * time.Second

var deleteChannelConfirmDurReadable = durafmt.Parse(deleteChannelConfirmDur).String()

func cmdAdminDeleteChannel(ctx context.Context, s *session, cmd string, args string) error {
	user, _ := splitSpace(args)
	user = cleanUsername(user)

	if user == "" {
		return s.ReplyUsage(ctx, "<user>")
	}

	if s.IRCChannel == user {
		return s.Replyf(ctx, "'%s' may not be deleted from their own channel. Run this command in another channel.", user)
	}

	channel, err := models.Channels(
		qm.Select(models.ChannelColumns.ID, models.ChannelColumns.BotName),
		models.ChannelWhere.Name.EQ(user),
	).One(ctx, s.Tx)
	if err != nil {
		if err == sql.ErrNoRows {
			return s.Replyf(ctx, "User '%s' does not exist.", user)
		}
		return err
	}

	confirmed, err := s.Confirm(ctx, s.User, "delete/"+user, deleteChannelConfirmDur)
	if err != nil {
		return err
	}

	if !confirmed {
		return s.Replyf(ctx, "If you are sure you want to delete channel '%s', run %s%s again in the next %s.", user, s.usageContext, user, deleteChannelConfirmDurReadable)
	}

	if err := modelsx.DeleteChannel(ctx, s.Tx, channel.ID); err != nil {
		return err
	}

	if err := s.Deps.Notifier.NotifyChannelUpdates(ctx, channel.BotName); err != nil {
		return err
	}

	return s.Replyf(ctx, "User '%s' has been deleted.", user)
}

func cmdAdminSleep(ctx context.Context, s *session, _ string, args string) error {
	usage := func() error {
		return s.ReplyUsage(ctx, "<dur>")
	}

	if args == "" {
		return usage()
	}

	dur, err := time.ParseDuration(args)
	if err != nil {
		return usage()
	}

	// Not using s.Deps.Clock here, since the testing framework is blocking and
	// cannot actually advance the clock while this is being handled.
	select {
	case <-time.After(dur):
	case <-ctx.Done():
		return ctx.Err()
	}

	return s.Replyf(ctx, "Slept for %s.", dur.String())
}

func cmdAdminSyncJoined(ctx context.Context, s *session, _ string, args string) error {
	botName, _ := splitSpace(args)
	if botName == "" {
		botName = s.Origin
	}

	if err := s.Deps.Notifier.NotifyChannelUpdates(ctx, strings.ToLower(botName)); err != nil {
		return err
	}

	return s.Replyf(ctx, "Triggered IRC channel sync for %s.", botName)
}
