package bot

import (
	"context"
	"database/sql"
	"strconv"
	"time"

	"github.com/hortbot/hortbot/internal/db/models"
	"github.com/hortbot/hortbot/internal/pkg/ctxlog"
	"github.com/volatiletech/sqlboiler/boil"
	"github.com/volatiletech/sqlboiler/queries/qm"
	"go.uber.org/zap"
)

func (b *Bot) updateRepeatedCommand(id int64, add bool, interval, wait time.Duration) {
	if add {
		b.rep.Add(id, b.runRepeatedCommand, interval, wait)
	} else {
		b.rep.Remove(id)
	}
}

func (b *Bot) runRepeatedCommand(ctx context.Context, id int64) {
	start := b.deps.Clock.Now()

	s := &session{
		Deps:      b.deps,
		Start:     start,
		UserLevel: levelEveryone,
	}

	ctx, logger := ctxlog.FromContextWith(ctx,
		zap.Int64("repeatedCommand", id),
	)

	err := transact(b.db, func(tx *sql.Tx) error {
		s.Tx = tx
		return handleRepeatedCommand(ctx, s, id)
	})

	if err != nil {
		logger.Warn("error running repeated command",
			zap.Error(err),
		)
	}
}

func handleRepeatedCommand(ctx context.Context, s *session, id int64) error {
	repeat, err := models.RepeatedCommands(
		models.RepeatedCommandWhere.ID.EQ(id),
		models.RepeatedCommandWhere.Enabled.EQ(true),
		qm.Load(models.RepeatedCommandRels.Channel),
		qm.Load(models.RepeatedCommandRels.SimpleCommand),
		qm.For("UPDATE"),
	).One(ctx, s.Tx)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil
		}
		return err
	}

	s.Channel = repeat.R.Channel
	s.Origin = s.Channel.BotName
	s.IRCChannel = s.Channel.Name
	s.RoomID = s.Channel.UserID
	s.RoomIDStr = strconv.FormatInt(s.RoomID, 10)

	s.N, err = s.messageCount()
	if err != nil {
		return err
	}

	if s.N < repeat.LastCount+repeat.MessageDiff {
		return nil
	}

	if ok, err := s.RepeatAllowed(id, repeat.Delay-1); !ok || err != nil {
		return err
	}

	repeat.LastCount = s.N

	if err := repeat.Update(ctx, s.Tx, boil.Whitelist(models.RepeatedCommandColumns.UpdatedAt, models.RepeatedCommandColumns.LastCount)); err != nil {
		return err
	}

	ctx, _ = ctxlog.FromContextWith(ctx,
		zap.Int64("roomID", s.RoomID),
		zap.String("channel", s.IRCChannel),
	)

	return runSimpleCommand(ctx, s, repeat.R.SimpleCommand)
}
