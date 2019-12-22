package bot

import (
	"context"
	"database/sql"
	"strconv"
	"time"

	"github.com/hortbot/hortbot/internal/db/models"
	"github.com/hortbot/hortbot/internal/pkg/ctxlog"
	"github.com/hortbot/hortbot/internal/pkg/repeat"
	"github.com/volatiletech/sqlboiler/boil"
	"github.com/volatiletech/sqlboiler/queries"
	"github.com/volatiletech/sqlboiler/queries/qm"
	"go.opencensus.io/trace"
	"go.uber.org/zap"
)

func (b *Bot) updateRepeatedCommand(id int64, add bool, interval, wait time.Duration) {
	if add {
		b.rep.Add(id, b.runRepeatedCommand, interval, wait)
	} else {
		b.rep.Remove(id)
	}
	setMetricRepeatGauges(b.rep)
}

func (b *Bot) updateScheduledCommand(id int64, add bool, expr *repeat.Cron) {
	if add {
		b.rep.AddCron(id, b.runScheduledCommand, expr)
	} else {
		b.rep.RemoveCron(id)
	}
	setMetricRepeatGauges(b.rep)
}

func (b *Bot) loadRepeats(ctx context.Context, reset bool) error {
	if reset {
		b.rep.Reset()
	}

	repeats, err := models.RepeatedCommands(
		models.RepeatedCommandWhere.Enabled.EQ(true),
	).All(ctx, b.db)
	if err != nil {
		return err
	}

	updateRepeating(b.deps, repeats, true)

	scheduleds, err := models.ScheduledCommands(
		models.ScheduledCommandWhere.Enabled.EQ(true),
	).All(ctx, b.db)
	if err != nil {
		return err
	}

	updateScheduleds(b.deps, scheduleds, true)

	return nil
}

func (b *Bot) runRepeatedCommand(ctx context.Context, id int64) {
	ctx, span := trace.StartSpan(ctx, "runRepeatedCommand")
	defer span.End()

	start := b.deps.Clock.Now()

	s := &session{
		Type:      sessionRepeat,
		Deps:      b.deps,
		Start:     start,
		UserLevel: levelEveryone,
	}

	ctx = ctxlog.With(ctx, zap.Int64("repeatedCommand", id))

	err := transact(ctx, b.db, func(ctx context.Context, tx *sql.Tx) error {
		s.Tx = tx
		return handleRepeatedCommand(ctx, s, id)
	})

	if err != nil {
		ctxlog.Warn(ctx, "error running repeated command", zap.Error(err))
	} else {
		metricRepeated.Inc()
	}
}

type repeatStatus struct {
	Enabled bool `boil:"enabled"`
	Active  bool `boil:"active"`
	Ready   bool `boil:"ready"`
}

func handleRepeatedCommand(ctx context.Context, s *session, id int64) error {
	ctx, span := trace.StartSpan(ctx, "handleRepeatedCommand")
	defer span.End()

	var status repeatStatus

	// Pre-check the status of the repeat and channel.
	err := queries.Raw(`
SELECT
	r.enabled AS enabled,
	c.active AS active,
	c.message_count >= (r.last_count + r.message_diff) AS ready
FROM
	repeated_commands r
JOIN
	channels c ON c.id = r.channel_id
WHERE
	r.id = $1
`, id).Bind(ctx, s.Tx, &status)
	if err != nil {
		if err == sql.ErrNoRows {
			status = repeatStatus{}
		} else {
			return err
		}
	}

	if !status.Enabled || !status.Active {
		s.Deps.UpdateRepeat(id, false, 0, 0)
		return nil
	}

	if !status.Ready {
		return nil
	}

	repeat, err := models.RepeatedCommands(
		models.RepeatedCommandWhere.ID.EQ(id),
		models.RepeatedCommandWhere.Enabled.EQ(true),
		qm.Load(models.RepeatedCommandRels.Channel),
		qm.Load(models.RepeatedCommandRels.CommandInfo, qm.For("UPDATE")),
		qm.Load(qm.Rels(models.RepeatedCommandRels.CommandInfo, models.CommandInfoRels.CustomCommand)),
		qm.Load(qm.Rels(models.RepeatedCommandRels.CommandInfo, models.CommandInfoRels.CommandList)),
		qm.For("UPDATE"),
	).One(ctx, s.Tx)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil
		}
		return err
	}

	if err := pgLock(ctx, s.Tx, repeat.R.Channel.UserID); err != nil {
		return err
	}

	if !repeat.R.Channel.Active {
		s.Deps.UpdateRepeat(id, false, 0, 0)
		return nil
	}

	repeatPopulateSession(ctx, s, repeat.R.Channel)

	if s.N < repeat.LastCount+repeat.MessageDiff {
		return nil
	}

	expiry := time.Duration(repeat.Delay-1) * time.Second
	if ok, err := s.RepeatAllowed(ctx, id, expiry); !ok || err != nil {
		return err
	}

	repeat.LastCount = s.N

	if err := repeat.Update(ctx, s.Tx, boil.Whitelist(models.RepeatedCommandColumns.LastCount)); err != nil {
		return err
	}

	info := repeat.R.CommandInfo

	info.Count++

	if err := info.Update(ctx, s.Tx, boil.Whitelist(models.CommandInfoColumns.Count)); err != nil {
		return err
	}

	ctx = ctxlog.With(ctx, zap.Int64("roomID", s.RoomID), zap.String("channel", s.IRCChannel))

	if command := info.R.CustomCommand; command != nil {
		return runCommandAndCount(ctx, s, info, command.Message, true)
	}

	items := info.R.CommandList.Items

	if len(items) == 0 {
		return nil
	}

	i := s.Deps.Rand.Intn(len(items))
	item := items[i]

	return runCommandAndCount(ctx, s, info, item, true)
}

func (b *Bot) runScheduledCommand(ctx context.Context, id int64) {
	ctx, span := trace.StartSpan(ctx, "runScheduledCommand")
	defer span.End()

	start := b.deps.Clock.Now()

	s := &session{
		Type:      sessionRepeat,
		Deps:      b.deps,
		Start:     start,
		UserLevel: levelEveryone,
	}

	ctx = ctxlog.With(ctx, zap.Int64("scheduledCommand", id))

	err := transact(ctx, b.db, func(ctx context.Context, tx *sql.Tx) error {
		s.Tx = tx
		return handleScheduledCommand(ctx, s, id)
	})

	if err != nil {
		ctxlog.Warn(ctx, "error running repeated command", zap.Error(err))
	} else {
		metricScheduled.Inc()
	}
}

func handleScheduledCommand(ctx context.Context, s *session, id int64) error {
	ctx, span := trace.StartSpan(ctx, "handleScheduledCommand")
	defer span.End()

	var status repeatStatus

	// Pre-check the status of the schedule and channel.
	err := queries.Raw(`
SELECT
	s.enabled AS enabled,
	c.active AS active,
	c.message_count >= (s.last_count + s.message_diff) AS ready
FROM
	scheduled_commands s
JOIN
	channels c ON c.id = s.channel_id
WHERE
	s.id = $1
`, id).Bind(ctx, s.Tx, &status)
	if err != nil {
		if err == sql.ErrNoRows {
			status = repeatStatus{}
		} else {
			return err
		}
	}

	if !status.Enabled || !status.Active {
		s.Deps.UpdateSchedule(id, false, nil)
		return nil
	}

	if !status.Ready {
		return nil
	}

	scheduled, err := models.ScheduledCommands(
		models.ScheduledCommandWhere.ID.EQ(id),
		models.ScheduledCommandWhere.Enabled.EQ(true),
		qm.Load(models.ScheduledCommandRels.Channel),
		qm.Load(models.ScheduledCommandRels.CommandInfo, qm.For("UPDATE")),
		qm.Load(qm.Rels(models.ScheduledCommandRels.CommandInfo, models.CommandInfoRels.CustomCommand)),
		qm.Load(qm.Rels(models.ScheduledCommandRels.CommandInfo, models.CommandInfoRels.CommandList)),
		qm.For("UPDATE"),
	).One(ctx, s.Tx)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil
		}
		return err
	}

	if err := pgLock(ctx, s.Tx, scheduled.R.Channel.UserID); err != nil {
		return err
	}

	if !scheduled.R.Channel.Active {
		s.Deps.UpdateSchedule(id, false, nil)
		return nil
	}

	repeatPopulateSession(ctx, s, scheduled.R.Channel)

	if s.N < scheduled.LastCount+scheduled.MessageDiff {
		return nil
	}

	// Hardcoded to 29 seconds, since cron jobs run at a fixed schedule
	// according to the clock rather than at an interval with an arbitrary
	// offset. This prevents any given cron from running faster than every
	// 30 seconds.
	if ok, err := s.ScheduledAllowed(ctx, id, 29*time.Second); !ok || err != nil {
		return err
	}

	scheduled.LastCount = s.N

	if err := scheduled.Update(ctx, s.Tx, boil.Whitelist(models.ScheduledCommandColumns.LastCount)); err != nil {
		return err
	}

	info := scheduled.R.CommandInfo

	info.Count++

	if err := info.Update(ctx, s.Tx, boil.Whitelist(models.CommandInfoColumns.Count)); err != nil {
		return err
	}

	ctx = ctxlog.With(ctx, zap.Int64("roomID", s.RoomID), zap.String("channel", s.IRCChannel))

	if command := info.R.CustomCommand; command != nil {
		return runCommandAndCount(ctx, s, info, command.Message, true)
	}

	items := info.R.CommandList.Items

	if len(items) == 0 {
		return nil
	}

	i := s.Deps.Rand.Intn(len(items))
	item := items[i]

	return runCommandAndCount(ctx, s, info, item, true)
}

func repeatPopulateSession(ctx context.Context, s *session, channel *models.Channel) {
	s.Channel = channel
	s.Origin = s.Channel.BotName
	s.IRCChannel = s.Channel.Name
	s.RoomID = s.Channel.UserID
	s.RoomIDStr = strconv.FormatInt(s.RoomID, 10)
	s.N = s.Channel.MessageCount
}

func updateRepeating(deps *sharedDeps, repeats []*models.RepeatedCommand, enable bool) {
	for _, repeat := range repeats {
		if !enable || !repeat.Enabled {
			deps.UpdateRepeat(repeat.ID, false, 0, 0)
			continue
		}

		delay := time.Duration(repeat.Delay) * time.Second
		delayNano := delay.Nanoseconds()

		start := repeat.UpdatedAt
		if repeat.InitTimestamp.Valid {
			start = repeat.InitTimestamp.Time
		}

		sinceUpdateNano := deps.Clock.Since(start).Nanoseconds()

		offsetNano := delayNano - sinceUpdateNano%delayNano
		offset := time.Duration(offsetNano) * time.Nanosecond

		deps.UpdateRepeat(repeat.ID, true, delay, offset)
	}
}

func updateScheduleds(deps *sharedDeps, scheduleds []*models.ScheduledCommand, enable bool) {
	for _, scheduled := range scheduleds {
		if !enable || !scheduled.Enabled {
			deps.UpdateSchedule(scheduled.ID, false, nil)
			continue
		}

		expr, err := repeat.ParseCron(scheduled.CronExpression)
		if err != nil {
			panic(err)
		}
		deps.UpdateSchedule(scheduled.ID, true, expr)
	}
}
