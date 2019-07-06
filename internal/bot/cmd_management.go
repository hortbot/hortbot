package bot

import (
	"context"
	"database/sql"
	"strconv"
	"strings"
	"time"

	"github.com/hako/durafmt"
	"github.com/hortbot/hortbot/internal/db/models"
	"github.com/volatiletech/sqlboiler/boil"
)

func handleManagement(ctx context.Context, s *session) error {
	var cmd string

	switch s.Message[0] {
	case '!', '+':
		cmd = s.Message[1:]
	}

	cmd = strings.ToLower(cmd)

	cmd, args := splitSpace(cmd)
	name, args := splitSpace(args)
	id, _ := splitSpace(args)

	name = strings.ToLower(name)

	switch cmd {
	case "join":
		return handleJoin(ctx, s, name, id)

	case "leave", "part":
		return handleLeave(ctx, s, name)
	}

	return nil
}

func handleJoin(ctx context.Context, s *session, name, id string) error {
	var channel *models.Channel
	var err error

	displayName := s.UserDisplay
	userID := s.UserID

	if name != "" && s.IsAdmin() {
		userID, err = strconv.ParseInt(id, 10, 64)
		if err != nil || userID <= 0 {
			return s.Replyf("Bad user ID: '%s'", id)
		}

		channel, err = models.Channels(models.ChannelWhere.Name.EQ(name)).One(ctx, s.Tx)
		displayName = name
	} else {
		channel, err = models.Channels(models.ChannelWhere.UserID.EQ(s.UserID)).One(ctx, s.Tx)
		name = s.User
	}

	if err != nil && err != sql.ErrNoRows {
		return err
	}

	botName := strings.TrimLeft(s.Origin, "#")

	if err == sql.ErrNoRows {
		channel = &models.Channel{
			UserID:               userID,
			Name:                 name,
			BotName:              botName,
			Active:               true,
			Prefix:               s.Deps.DefaultPrefix,
			ShouldModerate:       true,
			FilterCapsPercentage: 50,
			FilterCapsMinCaps:    6,
		}

		if err := channel.Insert(ctx, s.Tx, boil.Infer()); err != nil {
			return err
		}

		s.Deps.Notifier.NotifyChannelUpdates(channel.BotName)

		return s.Replyf("%s, %s will join your channel soon with prefix %s", displayName, botName, channel.Prefix)
	}

	if channel.Active {
		return s.Replyf("%s, %s is already active in your channel with prefix %s", displayName, channel.BotName, channel.Prefix)
	}

	channel.Active = true
	channel.BotName = botName

	if err := channel.Update(ctx, s.Tx, boil.Whitelist(models.ChannelColumns.UpdatedAt, models.ChannelColumns.Active, models.ChannelColumns.BotName)); err != nil {
		return err
	}

	s.Deps.Notifier.NotifyChannelUpdates(channel.BotName)

	return s.Replyf("%s, %s will join your channel soon with prefix %s", s.UserDisplay, channel.BotName, channel.Prefix)
}

func handleLeave(ctx context.Context, s *session, name string) error {
	var channel *models.Channel
	var err error

	displayName := s.UserDisplay

	if name != "" && s.IsAdmin() {
		channel, err = models.Channels(models.ChannelWhere.Name.EQ(name)).One(ctx, s.Tx)
		displayName = name
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

	s.Deps.Notifier.NotifyChannelUpdates(channel.BotName)

	return s.Replyf("%s, %s will now leave your channel.", displayName, channel.BotName)
}

const leaveConfirmSeconds = 10

var leaveConfirmReadable = durafmt.Parse(leaveConfirmSeconds * time.Second).String()

func cmdLeave(ctx context.Context, s *session, cmd string, args string) error {
	confirmed, err := s.Confirm(s.User, "leave", leaveConfirmSeconds)
	if err != nil {
		return err
	}

	if !confirmed {
		return s.Replyf("%s, if you are sure you want %s to leave this channel, run %s%s again in the next %s.", s.UserDisplay, s.Channel.BotName, s.Channel.Prefix, cmd, leaveConfirmReadable)
	}

	s.Channel.Active = false

	if err := s.Channel.Update(ctx, s.Tx, boil.Whitelist(models.ChannelColumns.UpdatedAt, models.ChannelColumns.Active)); err != nil {
		return err
	}

	s.Deps.Notifier.NotifyChannelUpdates(s.Channel.BotName)

	return s.Replyf("%s, %s will now leave your channel.", s.UserDisplay, s.Channel.BotName)
}
