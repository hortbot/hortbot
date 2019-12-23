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

func (b *Bot) runRepeatedCommand(ctx context.Context, id int64) {
	ctx, span := trace.StartSpan(ctx, "runRepeatedCommand")
	defer span.End()

	runner := &repeatedCommandRunner{
		id:   id,
		deps: b.deps,
	}
	if err := b.runRepeat(ctx, runner); err != nil {
		ctxlog.Warn(ctx, "error running repeated command", zap.Error(err))
	} else {
		metricRepeated.Inc()
	}
}

func (b *Bot) updateScheduledCommand(id int64, add bool, expr *repeat.Cron) {
	if add {
		b.rep.AddCron(id, b.runScheduledCommand, expr)
	} else {
		b.rep.RemoveCron(id)
	}
	setMetricRepeatGauges(b.rep)
}

func (b *Bot) runScheduledCommand(ctx context.Context, id int64) {
	ctx, span := trace.StartSpan(ctx, "runScheduledCommand")
	defer span.End()

	runner := &scheduledCommandRunner{
		id:   id,
		deps: b.deps,
	}
	if err := b.runRepeat(ctx, runner); err != nil {
		ctxlog.Warn(ctx, "error running scheduled command", zap.Error(err))
	} else {
		metricScheduled.Inc()
	}
}

type repeatRunner interface {
	withLog(ctx context.Context) context.Context
	status(ctx context.Context, exec boil.ContextExecutor) (status repeatStatus, err error)
	remove()
	load(ctx context.Context, exec boil.ContextExecutor) error
	channel() *models.Channel
	allowed(ctx context.Context) (bool, error)
	updateCount(ctx context.Context, exec boil.ContextExecutor) error
	info() *models.CommandInfo
}

type repeatStatus struct {
	Enabled bool `boil:"enabled"`
	Active  bool `boil:"active"`
	Ready   bool `boil:"ready"`
}

func (b *Bot) runRepeat(ctx context.Context, runner repeatRunner) error {
	ctx, span := trace.StartSpan(ctx, "runRepeat")
	defer span.End()

	ctx = runner.withLog(ctx)
	start := b.deps.Clock.Now()

	return transact(ctx, b.db, func(ctx context.Context, tx *sql.Tx) error {
		status, err := runner.status(ctx, tx)
		if err != nil {
			if err == sql.ErrNoRows {
				status = repeatStatus{}
			} else {
				return err
			}
		}

		if !status.Enabled || !status.Active {
			runner.remove()
			return nil
		}

		if !status.Ready {
			return nil
		}

		if err := runner.load(ctx, tx); err != nil {
			if err == sql.ErrNoRows {
				runner.remove()
				return nil
			}
			return err
		}

		channel := runner.channel()
		if err := pgLock(ctx, tx, channel.UserID); err != nil {
			return err
		}

		if allowed, err := runner.allowed(ctx); !allowed || err != nil {
			return err
		}

		if err := runner.updateCount(ctx, tx); err != nil {
			return err
		}

		s := &session{
			Type:       sessionRepeat,
			Deps:       b.deps,
			Tx:         tx,
			Start:      start,
			UserLevel:  levelEveryone,
			Channel:    channel,
			Origin:     channel.BotName,
			IRCChannel: channel.Name,
			RoomID:     channel.UserID,
			RoomIDStr:  strconv.FormatInt(channel.UserID, 10),
			N:          channel.MessageCount,
		}

		info := runner.info()

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
	})
}

type repeatedCommandRunner struct {
	id     int64
	deps   *sharedDeps
	repeat *models.RepeatedCommand
}

var _ repeatRunner = (*repeatedCommandRunner)(nil)

func (runner *repeatedCommandRunner) withLog(ctx context.Context) context.Context {
	return ctxlog.With(ctx, zap.Int64("repeatedCommand", runner.id))
}

func (runner *repeatedCommandRunner) status(ctx context.Context, exec boil.ContextExecutor) (status repeatStatus, err error) {
	err = queries.Raw(`
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
`, runner.id).Bind(ctx, exec, &status)
	return status, err
}

func (runner *repeatedCommandRunner) allowed(ctx context.Context) (bool, error) {
	channel := runner.channel()
	repeat := runner.repeat

	if !channel.Active || !repeat.Enabled {
		runner.remove()
		return false, nil
	}

	if channel.MessageCount < repeat.LastCount+repeat.MessageDiff {
		return false, nil
	}

	roomIDStr := strconv.FormatInt(channel.UserID, 10)
	expiry := time.Duration(repeat.Delay-1) * time.Second
	return runner.deps.Redis.RepeatAllowed(ctx, roomIDStr, runner.id, expiry)
}

func (runner *repeatedCommandRunner) remove() {
	runner.deps.UpdateRepeat(runner.id, false, 0, 0)
}

func (runner *repeatedCommandRunner) load(ctx context.Context, exec boil.ContextExecutor) error {
	repeat, err := models.RepeatedCommands(
		models.RepeatedCommandWhere.ID.EQ(runner.id),
		models.RepeatedCommandWhere.Enabled.EQ(true),
		qm.Load(models.RepeatedCommandRels.Channel),
		qm.Load(models.RepeatedCommandRels.CommandInfo, qm.For("UPDATE")),
		qm.Load(qm.Rels(models.RepeatedCommandRels.CommandInfo, models.CommandInfoRels.CustomCommand)),
		qm.Load(qm.Rels(models.RepeatedCommandRels.CommandInfo, models.CommandInfoRels.CommandList)),
		qm.For("UPDATE"),
	).One(ctx, exec)

	runner.repeat = repeat
	return err
}

func (runner *repeatedCommandRunner) channel() *models.Channel {
	return runner.repeat.R.Channel
}

func (runner *repeatedCommandRunner) updateCount(ctx context.Context, exec boil.ContextExecutor) error {
	repeat := runner.repeat
	repeat.LastCount = runner.channel().MessageCount
	return repeat.Update(ctx, exec, boil.Whitelist(models.RepeatedCommandColumns.LastCount))
}

func (runner *repeatedCommandRunner) info() *models.CommandInfo {
	return runner.repeat.R.CommandInfo
}

type scheduledCommandRunner struct {
	id        int64
	deps      *sharedDeps
	scheduled *models.ScheduledCommand
}

var _ repeatRunner = (*scheduledCommandRunner)(nil)

func (runner *scheduledCommandRunner) withLog(ctx context.Context) context.Context {
	return ctxlog.With(ctx, zap.Int64("scheduledCommand", runner.id))
}

func (runner *scheduledCommandRunner) status(ctx context.Context, exec boil.ContextExecutor) (status repeatStatus, err error) {
	err = queries.Raw(`
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
`, runner.id).Bind(ctx, exec, &status)
	return status, err
}

func (runner *scheduledCommandRunner) allowed(ctx context.Context) (bool, error) {
	channel := runner.channel()
	scheduled := runner.scheduled

	if !channel.Active || !scheduled.Enabled {
		runner.remove()
		return false, nil
	}

	if channel.MessageCount < scheduled.LastCount+scheduled.MessageDiff {
		return false, nil
	}

	// Hardcoded to 29 seconds, since cron jobs run at a fixed schedule
	// according to the clock rather than at an interval with an arbitrary
	// offset. This prevents any given cron from running faster than every
	// 30 seconds.
	roomIDStr := strconv.FormatInt(channel.UserID, 10)
	return runner.deps.Redis.ScheduledAllowed(ctx, roomIDStr, runner.id, 29*time.Second)
}

func (runner *scheduledCommandRunner) remove() {
	runner.deps.UpdateSchedule(runner.id, false, nil)
}

func (runner *scheduledCommandRunner) load(ctx context.Context, exec boil.ContextExecutor) error {
	scheduled, err := models.ScheduledCommands(
		models.ScheduledCommandWhere.ID.EQ(runner.id),
		models.ScheduledCommandWhere.Enabled.EQ(true),
		qm.Load(models.ScheduledCommandRels.Channel),
		qm.Load(models.ScheduledCommandRels.CommandInfo, qm.For("UPDATE")),
		qm.Load(qm.Rels(models.ScheduledCommandRels.CommandInfo, models.CommandInfoRels.CustomCommand)),
		qm.Load(qm.Rels(models.ScheduledCommandRels.CommandInfo, models.CommandInfoRels.CommandList)),
		qm.For("UPDATE"),
	).One(ctx, exec)

	runner.scheduled = scheduled
	return err
}

func (runner *scheduledCommandRunner) channel() *models.Channel {
	return runner.scheduled.R.Channel
}

func (runner *scheduledCommandRunner) updateCount(ctx context.Context, exec boil.ContextExecutor) error {
	scheduled := runner.scheduled
	scheduled.LastCount = runner.channel().MessageCount
	return scheduled.Update(ctx, exec, boil.Whitelist(models.ScheduledCommandColumns.LastCount))
}

func (runner *scheduledCommandRunner) info() *models.CommandInfo {
	return runner.scheduled.R.CommandInfo
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
