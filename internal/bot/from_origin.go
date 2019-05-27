package bot

import (
	"context"
	"database/sql"
	"strings"

	"github.com/hortbot/hortbot/internal/db/models"
	"github.com/volatiletech/sqlboiler/boil"
)

func handleFromOrigin(ctx context.Context, s *Session) error {
	var cmd string

	switch s.Message[0] {
	case '!', '+':
		cmd = s.Message[1:]
	}

	switch cmd {
	case "join":
		return handleJoin(ctx, s)

	case "leave", "part":
		return handleLeave(ctx, s)
	}

	return nil
}

func handleJoin(ctx context.Context, s *Session) error {
	channel, err := models.Channels(models.ChannelWhere.UserID.EQ(s.UserID)).One(ctx, s.Tx)
	if err != nil && err != sql.ErrNoRows {
		return err
	}

	botName := strings.TrimLeft(s.Origin, "#")

	if err == sql.ErrNoRows {
		channel = &models.Channel{
			UserID:  s.UserID,
			Name:    s.User,
			BotName: botName,
			Active:  true,
			Prefix:  s.Bot.prefix,
		}

		if err := channel.Insert(ctx, s.Tx, boil.Infer()); err != nil {
			return err
		}

		s.Notifier.NotifyChannelUpdates(channel.BotName)

		return s.Replyf("%s, %s will join your channel soon with prefix %s", s.UserDisplay, botName, channel.Prefix)
	}

	if channel.Active {
		return s.Replyf("%s, %s is already active in your channel with prefix %s", s.UserDisplay, channel.BotName, channel.Prefix)
	}

	channel.Active = true
	channel.BotName = botName

	if err := channel.Update(ctx, s.Tx, boil.Whitelist(models.ChannelColumns.UpdatedAt, models.ChannelColumns.Active, models.ChannelColumns.BotName)); err != nil {
		return err
	}

	s.Notifier.NotifyChannelUpdates(channel.BotName)

	return s.Replyf("%s, %s will join your channel soon with prefix %s", s.UserDisplay, channel.BotName, channel.Prefix)
}

func handleLeave(ctx context.Context, s *Session) error {
	channel, err := models.Channels(models.ChannelWhere.UserID.EQ(s.UserID)).One(ctx, s.Tx)
	if err != nil && err != sql.ErrNoRows {
		return err
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

	s.Notifier.NotifyChannelUpdates(channel.BotName)

	return s.Replyf("%s, %s will now leave your channel", s.UserDisplay, channel.BotName)
}
