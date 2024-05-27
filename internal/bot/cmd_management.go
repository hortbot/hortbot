package bot

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"slices"
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

func handleJoin(ctx context.Context, s *session, name string) error { //nolint:gocyclo
	displayName := s.UserDisplay
	userID := s.UserID

	name = cleanUsername(name)
	botName := strings.TrimLeft(s.Origin, "#")

	isAdmin := s.UserLevel.CanAccess(AccessLevelAdmin)
	adminOverride := false

	if name != "" && isAdmin {
		u, err := s.Deps.Twitch.GetUserByUsername(ctx, name)
		if err != nil {
			return s.Replyf(ctx, "Error getting ID from Twitch: %s", err.Error())
		}

		adminOverride = true
		userID = int64(u.ID)
		displayName = u.DispName()
	} else {
		if !isAdmin {
			replyDisabled := func() error {
				return s.Replyf(ctx, "Public join is disabled for %s; please contact an admin if you believe this to be an error.", botName)
			}

			if !s.Deps.PublicJoin {
				return replyDisabled()
			}

			if _, ok := stringSliceIndex(s.Deps.PublicJoinDisabled, botName); ok {
				return replyDisabled()
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
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return err
	}

	hasToken := true
	tt, authErr := models.TwitchTokens(models.TwitchTokenWhere.TwitchID.EQ(userID)).One(ctx, s.Tx)
	if authErr != nil {
		if !errors.Is(authErr, sql.ErrNoRows) {
			return authErr
		}
		hasToken = false
	}
	hasBotScope := hasToken && slices.Contains(tt.Scopes, "channel:bot")

	noAuth := !hasBotScope
	if noAuth {
		isModerator, err := models.ModeratedChannels(models.ModeratedChannelWhere.BroadcasterID.EQ(userID), models.ModeratedChannelWhere.BotName.EQ(botName)).Exists(ctx, s.Tx)
		if err != nil {
			return err
		}
		noAuth = !isModerator
	}

	if noAuth {
		if adminOverride {
			return s.Replyf(ctx, "The can no longer join channels without auth.")
		}

		if channel != nil && channel.Active {
			return s.Replyf(ctx, "Due to Twitch policy changes, you must explicitly allow the bot to rejoin your chat. Please login at %s/login and return here.", s.WebAddrFor(botName))
		}

		return s.Replyf(ctx, "Thanks for your interest; before I can join your channel, you need to log in to the website to give me permission to join your chat. Please login at %s/login and return here.", s.WebAddrFor(botName))
	}

	firstJoin := func(ctx context.Context) error {
		return s.Replyf(ctx, "%s, %s will join your channel soon with prefix '%s'.", displayName, botName, channel.Prefix)
	}

	if errors.Is(err, sql.ErrNoRows) {
		channel = modelsx.NewChannel(userID, name, displayName, botName)

		if err := channel.Insert(ctx, s.Tx, boil.Infer()); err != nil {
			return err
		}

		if err := s.Deps.EventsubUpdateNotifier.NotifyEventsubUpdates(ctx); err != nil {
			return fmt.Errorf("notify eventsub updates: %w", err)
		}

		return firstJoin(ctx)
	}

	if channel.Active {
		if channel.Name == name {
			return s.Replyf(ctx, "%s, %s is already active in your channel with prefix '%s'. If the bot isn't responding and your channel is in follower-only mode, ensure you've modded the bot.", displayName, channel.BotName, channel.Prefix)
		}

		channel.Name = name
		channel.DisplayName = displayName

		if err := channel.Update(ctx, s.Tx, boil.Whitelist(models.ChannelColumns.UpdatedAt, models.ChannelColumns.Name, models.ChannelColumns.DisplayName)); err != nil {
			return err
		}

		if err := s.Deps.EventsubUpdateNotifier.NotifyEventsubUpdates(ctx); err != nil {
			return fmt.Errorf("notify eventsub updates: %w", err)
		}

		return s.Replyf(ctx, "%s, %s will now rejoin your channel with your new username.", displayName, channel.BotName)
	}

	channel.Active = true
	channel.BotName = botName
	channel.Name = name
	channel.DisplayName = displayName

	if err := channel.Update(ctx, s.Tx, boil.Whitelist(models.ChannelColumns.UpdatedAt, models.ChannelColumns.Active, models.ChannelColumns.BotName, models.ChannelColumns.Name, models.ChannelColumns.DisplayName)); err != nil {
		return err
	}

	if err := s.Deps.EventsubUpdateNotifier.NotifyEventsubUpdates(ctx); err != nil {
		return fmt.Errorf("notify eventsub updates: %w", err)
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

	if name != "" && s.UserLevel.CanAccess(AccessLevelAdmin) {
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

	if errors.Is(err, sql.ErrNoRows) {
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

	if err := s.Deps.EventsubUpdateNotifier.NotifyEventsubUpdates(ctx); err != nil {
		return fmt.Errorf("notify eventsub updates: %w", err)
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

	if err := s.Deps.EventsubUpdateNotifier.NotifyEventsubUpdates(ctx); err != nil {
		return fmt.Errorf("notify eventsub updates: %w", err)
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
