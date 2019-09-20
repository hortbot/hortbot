package bot

import (
	"context"
	"database/sql"
	"strings"
	"time"

	"github.com/hako/durafmt"
	"github.com/hortbot/hortbot/internal/db/models"
	"github.com/hortbot/hortbot/internal/db/modelsx"
	"github.com/hortbot/hortbot/internal/pkg/ctxlog"
	"github.com/volatiletech/sqlboiler/boil"
	"go.opencensus.io/trace"
	"go.uber.org/zap"
)

func handleManagement(ctx context.Context, s *session) error {
	ctx, span := trace.StartSpan(ctx, "handleManagement")
	defer span.End()

	var cmd string

	switch s.Message[0] {
	case '!', '+':
		cmd = s.Message[1:]
	}

	cmd = strings.ToLower(cmd)
	cmd, args := splitSpace(cmd)

	switch cmd {
	case "join":
		return handleJoin(ctx, s, args)

	case "leave", "part":
		return handleLeave(ctx, s, args)

	case "admin":
		return cmdAdmin(ctx, s, cmd, args)
	}

	return nil
}

func handleJoin(ctx context.Context, s *session, name string) error {
	displayName := s.UserDisplay
	userID := s.UserID

	name = cleanUsername(name)

	if name != "" && s.IsAdmin() {
		u, err := s.Deps.Twitch.GetUserForUsername(ctx, name)
		if err != nil {
			return s.Replyf(ctx, "Error getting ID from Twitch: %s", err.Error())
		}

		userID = u.ID
		displayName = u.DispName()
	} else {
		name = s.User
	}

	blocked, err := models.BlockedUsers(models.BlockedUserWhere.TwitchID.EQ(userID)).Exists(ctx, s.Tx)
	if err != nil {
		return err
	}

	if blocked {
		ctxlog.FromContext(ctx).Warn("user is blocked", zap.String("name", name), zap.Int64("user_id", userID))
		return nil
	}

	channel, err := models.Channels(models.ChannelWhere.UserID.EQ(userID)).One(ctx, s.Tx)
	if err != nil && err != sql.ErrNoRows {
		return err
	}

	botName := strings.TrimLeft(s.Origin, "#")

	if err == sql.ErrNoRows {
		channel = modelsx.NewChannel(userID, name, botName)

		if err := channel.Insert(ctx, s.Tx, boil.Infer()); err != nil {
			return err
		}

		if err := s.Deps.Notifier.NotifyChannelUpdates(ctx, channel.BotName); err != nil {
			return err
		}

		return s.Replyf(ctx, "%s, %s will join your channel soon with prefix %s", displayName, botName, channel.Prefix)
	}

	if channel.Active {
		return s.Replyf(ctx, "%s, %s is already active in your channel with prefix %s", displayName, channel.BotName, channel.Prefix)
	}

	channel.Active = true
	channel.BotName = botName

	if err := channel.Update(ctx, s.Tx, boil.Whitelist(models.ChannelColumns.UpdatedAt, models.ChannelColumns.Active, models.ChannelColumns.BotName)); err != nil {
		return err
	}

	if err := s.Deps.Notifier.NotifyChannelUpdates(ctx, channel.BotName); err != nil {
		return err
	}

	return s.Replyf(ctx, "%s, %s will join your channel soon with prefix %s", displayName, channel.BotName, channel.Prefix)
}

func handleLeave(ctx context.Context, s *session, name string) error {
	var channel *models.Channel
	var err error

	name = cleanUsername(name)

	displayName := s.UserDisplay

	if name != "" && s.IsAdmin() {
		channel, err = models.Channels(models.ChannelWhere.Name.EQ(name)).One(ctx, s.Tx)
		displayName = name // TODO: Fetch this from the database when the column exists.
	} else {
		channel, err = models.Channels(models.ChannelWhere.UserID.EQ(s.UserID)).One(ctx, s.Tx)
	}

	if err == sql.ErrNoRows {
		return nil
	}

	if !channel.Active {
		return nil
	}

	channel.Active = false

	if err := channel.Update(ctx, s.Tx, boil.Whitelist(models.ChannelColumns.UpdatedAt, models.ChannelColumns.Active)); err != nil {
		return err
	}

	if err := s.Deps.Notifier.NotifyChannelUpdates(ctx, channel.BotName); err != nil {
		return err
	}

	return s.Replyf(ctx, "%s, %s will now leave your channel.", displayName, channel.BotName)
}

const leaveConfirmDur = 10 * time.Second

var leaveConfirmDurReadable = durafmt.Parse(leaveConfirmDur).String()

func cmdLeave(ctx context.Context, s *session, cmd string, args string) error {
	confirmed, err := s.Confirm(ctx, s.User, "leave", leaveConfirmDur)
	if err != nil {
		return err
	}

	if !confirmed {
		return s.Replyf(ctx, "%s, if you are sure you want %s to leave this channel, run %s%s again in the next %s.", s.UserDisplay, s.Channel.BotName, s.Channel.Prefix, cmd, leaveConfirmDurReadable)
	}

	s.Channel.Active = false

	if err := s.Channel.Update(ctx, s.Tx, boil.Whitelist(models.ChannelColumns.UpdatedAt, models.ChannelColumns.Active)); err != nil {
		return err
	}

	if err := s.Deps.Notifier.NotifyChannelUpdates(ctx, s.Channel.BotName); err != nil {
		return err
	}

	return s.Replyf(ctx, "%s, %s will now leave your channel.", s.UserDisplay, s.Channel.BotName)
}
