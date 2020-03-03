package bot

import (
	"context"
	"database/sql"
	"strconv"
	"strings"
	"time"

	"github.com/hortbot/hortbot/internal/db/models"
	"github.com/hortbot/hortbot/internal/pkg/ctxlog"
	"github.com/hortbot/hortbot/internal/pkg/dbx"
	"github.com/jakebailey/irc"
	"github.com/volatiletech/sqlboiler/boil"
	"github.com/volatiletech/sqlboiler/queries/qm"
	"go.opencensus.io/trace"
	"go.uber.org/zap"
)

func (b *Bot) handleNotice(ctx context.Context, origin string, m *irc.Message) error {
	ctx, span := trace.StartSpan(ctx, "handleNotice")
	defer span.End()

	if len(m.Tags) == 0 || len(m.Params) == 0 {
		return errInvalidMessage
	}

	msgID := m.Tags["msg-id"]
	ircChannel := strings.TrimLeft(m.Params[0], "#")

	ctx = ctxlog.With(ctx, zap.String("msg_id", msgID), zap.String("channel", ircChannel))

	switch msgID {
	case "msg_followersonly", "msg_followersonly_zero":
		return b.handleNoticeFollow(ctx, origin, ircChannel)
	case "msg_followersonly_followed":
		return nil
	case "msg_banned", "msg_channel_suspended":
		return b.handleNoticeLeaveChannel(ctx, origin, ircChannel)
	}

	ctxlog.Debug(ctx, "unhandled NOTICE", zap.Any("message", m))

	return nil
}

func (b *Bot) handleNoticeFollow(ctx context.Context, origin string, ircChannel string) error {
	return dbx.Transact(ctx, b.db,
		dbx.SetLocalLockTimeout(5*time.Second),
		func(ctx context.Context, tx *sql.Tx) error {
			channel, err := models.Channels(
				qm.Select(models.ChannelColumns.UserID),
				models.ChannelWhere.Name.EQ(ircChannel),
				models.ChannelWhere.BotName.EQ(origin),
			).One(ctx, tx)
			if err != nil {
				if err == sql.ErrNoRows {
					ctxlog.Warn(ctx, "received follower-only message for unknown user")
					return nil
				}
				return err
			}

			seen, err := b.deps.Redis.CheckAndMarkCooldown(ctx, strconv.FormatInt(channel.UserID, 10), "follow_cooldown", 10*time.Minute)
			if err != nil || seen {
				return err
			}

			if err := followUser(ctx, tx, b.deps.Twitch, origin, channel.UserID); err != nil {
				ctxlog.Warn(ctx, "error following user", zap.Error(err))
			}

			return nil
		})
}

func (b *Bot) handleNoticeLeaveChannel(ctx context.Context, origin string, ircChannel string) error {
	ctxlog.Info(ctx, "bot was banned or the channel is suspended, leaving")

	return dbx.Transact(ctx, b.db,
		dbx.SetLocalLockTimeout(5*time.Second),
		func(ctx context.Context, tx *sql.Tx) error {
			channel, err := models.Channels(
				qm.Select(models.ChannelColumns.ID, models.ChannelColumns.UpdatedAt),
				models.ChannelWhere.Name.EQ(ircChannel),
				models.ChannelWhere.BotName.EQ(origin),
			).One(ctx, tx)
			if err != nil {
				if err == sql.ErrNoRows {
					ctxlog.Warn(ctx, "received ban notice for unknown user")
					return nil
				}
				return err
			}

			channel.Active = false

			if err := channel.Update(ctx, tx, boil.Whitelist(models.ChannelColumns.UpdatedAt, models.ChannelColumns.Active)); err != nil {
				return err
			}

			return b.deps.Notifier.NotifyChannelUpdates(ctx, origin)
		})
}
