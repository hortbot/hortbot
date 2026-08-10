package bot

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/hako/durafmt"
	"github.com/hortbot/hortbot/internal/db/dbsql"
	"github.com/jackc/pgx/v5"
	"github.com/zikaeroh/ctxlog"
	"go.uber.org/zap"
)

func handleManagement(ctx context.Context, s *session) error {
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
	botName := strings.TrimLeft(s.BotLogin, "#")

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

	q := s.Queries
	blocked, err := q.IsBlockedUser(ctx, userID)
	if err != nil {
		return fmt.Errorf("checking blocked users: %w", err)
	}

	if blocked {
		ctxlog.Warn(ctx, "user is blocked", zap.String("name", name), zap.Int64("user_id", userID))
		return nil
	}

	channelRow, err := q.GetChannelByTwitchIDForUpdate(ctx, userID)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("getting channel: %w", err)
	}
	var channel *dbsql.Channel
	if err == nil {
		channel = &channelRow
	}

	hasToken := true
	tt, authErr := q.GetTwitchTokenByID(ctx, userID)
	if authErr != nil {
		if !errors.Is(authErr, pgx.ErrNoRows) {
			return fmt.Errorf("getting token: %w", authErr)
		}
		hasToken = false
	}
	hasBotScope := hasToken && slices.Contains(tt.Scopes, "channel:bot")

	noAuth := !hasBotScope
	if noAuth {
		isModerator, err := q.IsModeratedChannel(ctx, dbsql.IsModeratedChannelParams{
			BroadcasterID: userID,
			BotName:       botName,
		})
		if err != nil {
			return fmt.Errorf("checking moderated channels: %w", err)
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

	if errors.Is(err, pgx.ErrNoRows) {
		inserted, err := q.InsertDefaultChannel(ctx, dbsql.InsertDefaultChannelParams{
			TwitchID: userID, Name: name, DisplayName: displayName, BotName: botName,
		})
		if err != nil {
			return fmt.Errorf("inserting channel: %w", err)
		}
		channel = &inserted

		s.requestEventsubUpdate()

		return firstJoin(ctx)
	}

	if channel.Active {
		if channel.Name == name {
			s.requestEventsubUpdate()
			return s.Replyf(ctx, "%s, %s is already configured for your channel with prefix '%s'. I've requested a reconnect.", displayName, channel.BotName, channel.Prefix)
		}

		channel.Name = name
		channel.DisplayName = displayName

		if err := q.SaveChannelMembership(ctx, channel); err != nil {
			return fmt.Errorf("updating channel: %w", err)
		}

		s.requestEventsubUpdate()

		return s.Replyf(ctx, "%s, %s will now rejoin your channel with your new username.", displayName, channel.BotName)
	}

	channel.Active = true
	channel.BotName = botName
	channel.Name = name
	channel.DisplayName = displayName

	if err := q.SaveChannelMembership(ctx, channel); err != nil {
		return fmt.Errorf("updating channel: %w", err)
	}

	s.requestEventsubUpdate()

	repeated, err := q.ListRepeatedCommands(ctx, channel.ID)
	if err != nil {
		return fmt.Errorf("getting repeated commands: %w", err)
	}
	if err := updateRepeating(ctx, s.Deps, repeated, true); err != nil {
		return err
	}

	scheduled, err := q.ListScheduledCommands(ctx, channel.ID)
	if err != nil {
		return fmt.Errorf("getting scheduled commands: %w", err)
	}
	if err := updateScheduleds(ctx, s.Deps, scheduled, true); err != nil {
		return err
	}

	return firstJoin(ctx)
}

func handleLeave(ctx context.Context, s *session, name string) error {
	var channel *dbsql.Channel
	var err error

	name = cleanUsername(name)

	displayName := name

	if name != "" && s.UserLevel.CanAccess(AccessLevelAdmin) {
		row, queryErr := s.Queries.GetChannelByNameForUpdate(ctx, name)
		err = queryErr
		if queryErr == nil {
			channel = &row
		}
	} else {
		displayName = s.UserDisplay
		row, queryErr := s.Queries.GetChannelByTwitchIDForUpdate(ctx, s.UserID)
		err = queryErr
		if queryErr == nil {
			channel = &row
		}
	}

	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}

	if err != nil {
		return fmt.Errorf("getting channel: %w", err)
	}

	if !channel.Active {
		return nil
	}

	if name != "" && channel.DisplayName != "" {
		displayName = channel.DisplayName
	}

	channel.Active = false

	q := s.Queries
	if err := q.UpdateChannelActive(ctx, dbsql.UpdateChannelActiveParams{Active: false, ID: channel.ID}); err != nil {
		return fmt.Errorf("updating channel: %w", err)
	}

	s.requestEventsubUpdate()

	repeated, err := q.ListRepeatedCommands(ctx, channel.ID)
	if err != nil {
		return fmt.Errorf("getting repeated commands: %w", err)
	}
	if err := updateRepeating(ctx, s.Deps, repeated, false); err != nil {
		return err
	}

	scheduled, err := q.ListScheduledCommands(ctx, channel.ID)
	if err != nil {
		return fmt.Errorf("getting scheduled commands: %w", err)
	}
	if err := updateScheduleds(ctx, s.Deps, scheduled, false); err != nil {
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

	q := s.Queries
	if err := q.UpdateChannelActive(ctx, dbsql.UpdateChannelActiveParams{Active: false, ID: s.Channel.ID}); err != nil {
		return fmt.Errorf("updating channel: %w", err)
	}

	s.requestEventsubUpdate()

	repeated, err := q.ListRepeatedCommands(ctx, s.Channel.ID)
	if err != nil {
		return fmt.Errorf("getting repeated commands: %w", err)
	}

	if err := updateRepeating(ctx, s.Deps, repeated, false); err != nil {
		return err
	}

	scheduleds, err := q.ListScheduledCommands(ctx, s.Channel.ID)
	if err != nil {
		return fmt.Errorf("getting scheduled commands: %w", err)
	}

	if err := updateScheduleds(ctx, s.Deps, scheduleds, false); err != nil {
		return err
	}

	return s.Replyf(ctx, "%s, %s will now leave your channel.", s.UserDisplay, s.Channel.BotName)
}
