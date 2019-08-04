package bot

import (
	"context"
	"database/sql"
	"strings"

	"github.com/hortbot/hortbot/internal/db/models"
	"github.com/volatiletech/sqlboiler/boil"
)

var adminCommands = newHandlerMap(map[string]handlerFunc{
	"roundtrip": {fn: cmdAdminRoundtrip, minLevel: levelAdmin},
	"block":     {fn: cmdAdminBlock, minLevel: levelAdmin},
	"unblock":   {fn: cmdAdminUnblock, minLevel: levelAdmin},
})

func cmdAdmin(ctx context.Context, s *session, cmd string, args string) error {
	subcommand, args := splitSpace(args)
	subcommand = strings.ToLower(subcommand)

	ok, err := adminCommands.run(ctx, s, subcommand, args)
	if err != nil {
		return err
	}

	if !ok {
		return s.Replyf("Bad command %s", subcommand)
	}

	return nil
}

func cmdAdminRoundtrip(ctx context.Context, s *session, cmd string, args string) error {
	now := s.Deps.Clock.Now()
	return s.Replyf("total=%v, handle=%v", now.Sub(s.TMISent), now.Sub(s.Start))
}

func cmdAdminBlock(ctx context.Context, s *session, cmd string, args string) error {
	if args == "" {
		return s.ReplyUsage("<username>")
	}

	id, err := s.Deps.Twitch.GetIDForUsername(ctx, args)
	if err != nil {
		return s.Replyf("Error getting ID from Twitch: %s", err.Error())
	}

	bu := &models.BlockedUser{TwitchID: id}
	if err := bu.Upsert(ctx, s.Tx, false, []string{models.BlockedUserColumns.TwitchID}, boil.Infer(), boil.Infer()); err != nil {
		return err
	}

	channel, err := models.Channels(models.ChannelWhere.UserID.EQ(id)).One(ctx, s.Tx)
	if err != nil && err != sql.ErrNoRows {
		return err
	}

	if err != sql.ErrNoRows && channel.Active {
		channel.Active = false

		if err := channel.Update(ctx, s.Tx, boil.Whitelist(models.ChannelColumns.UpdatedAt, models.ChannelColumns.Active)); err != nil {
			return err
		}

		s.Deps.Notifier.NotifyChannelUpdates(channel.BotName)
	}

	return s.Replyf("%s (%d) has been blocked.", args, id)
}

func cmdAdminUnblock(ctx context.Context, s *session, cmd string, args string) error {
	if args == "" {
		return s.ReplyUsage("<username>")
	}

	id, err := s.Deps.Twitch.GetIDForUsername(ctx, args)
	if err != nil {
		return s.Replyf("Error getting ID from Twitch: %s", err.Error())
	}

	if err := models.BlockedUsers(models.BlockedUserWhere.TwitchID.EQ(id)).DeleteAll(ctx, s.Tx); err != nil {
		return err
	}

	return s.Replyf("%s (%d) has been unblocked.", args, id)
}
