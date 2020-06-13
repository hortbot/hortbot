package bot

import (
	"context"
	"database/sql"
	"strings"
	"time"

	"github.com/hako/durafmt"
	"github.com/hortbot/hortbot/internal/db/models"
	"github.com/hortbot/hortbot/internal/db/modelsx"
	"github.com/volatiletech/sqlboiler/v4/boil"
	"github.com/volatiletech/sqlboiler/v4/queries/qm"
	"github.com/zikaeroh/ctxlog"
	"go.opencensus.io/trace"
	"go.uber.org/zap"
)

func handleManagement(ctx context.Context, s *session) error {
	ctx, span := trace.StartSpan(ctx, "handleManagement")
	defer span.End()

	prefix := s.Message[0]
	switch prefix {
	case '!', '+':
	default:
		return nil
	}

	cmd, args := splitSpace(s.Message[1:])
	cmd = cleanCommandName(cmd)

	defer s.UsageContext(string(prefix) + cmd)()

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
	botName := strings.TrimLeft(s.Origin, "#")

	isAdmin := s.UserLevel.CanAccess(levelAdmin)

	if name != "" && isAdmin {
		u, err := s.Deps.Twitch.GetUserForUsername(ctx, name)
		if err != nil {
			return s.Replyf(ctx, "Error getting ID from Twitch: %s", err.Error())
		}

		userID = u.ID
		displayName = u.DispName()
	} else {
		if !isAdmin {
			if !s.Deps.PublicJoin {
				return nil
			}

			if _, ok := stringSliceIndex(s.Deps.PublicJoinBlacklist, botName); ok {
				return nil
			}
		}

		name = s.User
	}

	blocked, err := models.BlockedUsers(models.BlockedUserWhere.TwitchID.EQ(userID)).Exists(ctx, s.Tx)
	if err != nil {
		return err
	}

	if blocked {
		ctxlog.Warn(ctx, "user is blocked", zap.String("name", name), zap.Int64("user_id", userID))
		return nil
	}

	channel, err := models.Channels(
		models.ChannelWhere.TwitchID.EQ(userID),
		qm.Load(models.ChannelRels.RepeatedCommands),
		qm.Load(models.ChannelRels.ScheduledCommands),
		qm.For("UPDATE"),
	).One(ctx, s.Tx)
	if err != nil && err != sql.ErrNoRows {
		return err
	}

	firstJoin := func(ctx context.Context) error {
		return s.Replyf(ctx, "%s, %s will join your channel soon with prefix '%s'. Log in to the website to give the bot permission to access your Twitch account: %s/login", displayName, botName, channel.Prefix, s.WebAddrFor(botName))
	}

	if err == sql.ErrNoRows {
		channel = modelsx.NewChannel(userID, name, displayName, botName)

		if err := channel.Insert(ctx, s.Tx, boil.Infer()); err != nil {
			return err
		}

		if err := s.Deps.Notifier.NotifyChannelUpdates(ctx, channel.BotName); err != nil {
			return err
		}

		return firstJoin(ctx)
	}

	if channel.Active {
		if channel.Name == name {
			return s.Replyf(ctx, "%s, %s is already active in your channel with prefix '%s'.", displayName, channel.BotName, channel.Prefix)
		}

		channel.Name = name
		channel.DisplayName = displayName

		if err := channel.Update(ctx, s.Tx, boil.Whitelist(models.ChannelColumns.UpdatedAt, models.ChannelColumns.Name, models.ChannelColumns.DisplayName)); err != nil {
			return err
		}

		if err := s.Deps.Notifier.NotifyChannelUpdates(ctx, channel.BotName); err != nil {
			return err
		}

		return s.Replyf(ctx, "%s, %s will now rejoin your channel with your new username.", displayName, channel.BotName)
	}

	channel.Active = true
	channel.BotName = botName

	if err := channel.Update(ctx, s.Tx, boil.Whitelist(models.ChannelColumns.UpdatedAt, models.ChannelColumns.Active, models.ChannelColumns.BotName)); err != nil {
		return err
	}

	if err := s.Deps.Notifier.NotifyChannelUpdates(ctx, channel.BotName); err != nil {
		return err
	}

	if err := updateRepeating(ctx, s.Deps, channel.R.RepeatedCommands, true); err != nil {
		return err
	}

	if err := updateScheduleds(ctx, s.Deps, channel.R.ScheduledCommands, true); err != nil {
		return err
	}

	return firstJoin(ctx)
}

func handleLeave(ctx context.Context, s *session, name string) error {
	var channel *models.Channel
	var err error

	name = cleanUsername(name)

	displayName := name

	if name != "" && s.UserLevel.CanAccess(levelAdmin) {
		channel, err = models.Channels(
			models.ChannelWhere.Name.EQ(name),
			qm.Load(models.ChannelRels.RepeatedCommands),
			qm.Load(models.ChannelRels.ScheduledCommands),
			qm.For("UPDATE"),
		).One(ctx, s.Tx)
	} else {
		displayName = s.UserDisplay
		channel, err = models.Channels(
			models.ChannelWhere.TwitchID.EQ(s.UserID),
			qm.Load(models.ChannelRels.RepeatedCommands),
			qm.Load(models.ChannelRels.ScheduledCommands),
			qm.For("UPDATE"),
		).One(ctx, s.Tx)
	}

	if err == sql.ErrNoRows {
		return nil
	}

	if err != nil {
		return err
	}

	if !channel.Active {
		return nil
	}

	if name != "" && channel.DisplayName != "" {
		displayName = channel.DisplayName
	}

	channel.Active = false

	if err := channel.Update(ctx, s.Tx, boil.Whitelist(models.ChannelColumns.UpdatedAt, models.ChannelColumns.Active)); err != nil {
		return err
	}

	if err := s.Deps.Notifier.NotifyChannelUpdates(ctx, channel.BotName); err != nil {
		return err
	}

	if err := updateRepeating(ctx, s.Deps, channel.R.RepeatedCommands, false); err != nil {
		return err
	}

	if err := updateScheduleds(ctx, s.Deps, channel.R.ScheduledCommands, false); err != nil {
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

	repeated, err := s.Channel.RepeatedCommands().All(ctx, s.Tx)
	if err != nil {
		return err
	}

	if err := updateRepeating(ctx, s.Deps, repeated, false); err != nil {
		return err
	}

	scheduleds, err := s.Channel.ScheduledCommands().All(ctx, s.Tx)
	if err != nil {
		return err
	}

	if err := updateScheduleds(ctx, s.Deps, scheduleds, false); err != nil {
		return err
	}

	return s.Replyf(ctx, "%s, %s will now leave your channel.", s.UserDisplay, s.Channel.BotName)
}
