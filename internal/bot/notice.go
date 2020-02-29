package bot

import (
	"context"
	"database/sql"
	"strconv"
	"strings"
	"time"

	"github.com/hortbot/hortbot/internal/db/models"
	"github.com/hortbot/hortbot/internal/pkg/ctxlog"
	"github.com/jakebailey/irc"
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
	ctx = ctxlog.With(ctx, zap.String("msg_id", msgID))

	if !strings.HasPrefix(msgID, "msg_followersonly") {
		ctxlog.Debug(ctx, "unhandled command", zap.Any("message", m))
		return nil
	}

	if msgID == "msg_followersonly_followed" {
		// Already followed, no need to follow again.
		return nil
	}

	username := strings.TrimLeft(m.Params[0], "#")
	ctx = ctxlog.With(ctx, zap.String("channel", m.Params[0]))

	return transact(ctx, b.db, func(ctx context.Context, tx *sql.Tx) error {
		channel, err := models.Channels(models.ChannelWhere.Name.EQ(username), qm.Select(models.ChannelColumns.UserID)).One(ctx, tx)
		if err != nil {
			if err == sql.ErrNoRows {
				ctxlog.Warn(ctx, "received follower-only message for unknown user")
				return nil
			}
			return err
		}

		seen, err := b.deps.Redis.CheckAndMarkCooldown(ctx, strconv.FormatInt(channel.UserID, 10), "follow_cooldown", 10*time.Minute)
		if err != nil {
			return err
		}

		if seen {
			return nil
		}

		if err := followUser(ctx, tx, b.deps.Twitch, origin, channel.UserID); err != nil {
			ctxlog.Warn(ctx, "error following user", zap.Error(err))
		}

		return nil
	})
}
