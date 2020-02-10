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

func (b *Bot) addRepeat(ctx context.Context, id int64, start time.Time, interval time.Duration) error {
	defer setMetricRepeatGauges(ctx, b.rep)
	return b.rep.Add(ctx, id, b.runRepeatedCommand, start, interval)
}

func (b *Bot) removeRepeat(ctx context.Context, id int64) error {
	defer setMetricRepeatGauges(ctx, b.rep)
	return b.rep.Remove(ctx, id)
}

func (b *Bot) runRepeatedCommand(ctx context.Context, id int64) (readd bool) {
	ctx, span := trace.StartSpan(ctx, "runRepeatedCommand")
	defer span.End()

	runner := &repeatedCommandRunner{
		id:   id,
		deps: b.deps,
	}

	readd, err := b.runRepeat(ctx, runner)
	if err != nil {
		ctxlog.Warn(ctx, "error running repeated command", zap.Error(err))
	} else {
		metricRepeated.Inc()
	}
	return readd
}

func (b *Bot) addScheduled(ctx context.Context, id int64, expr *repeat.Cron) error {
	defer setMetricRepeatGauges(ctx, b.rep)
	return b.rep.AddCron(ctx, id, b.runScheduledCommand, expr)
}

func (b *Bot) removeScheduled(ctx context.Context, id int64) error {
	defer setMetricRepeatGauges(ctx, b.rep)
	return b.rep.RemoveCron(ctx, id)
}

func (b *Bot) runScheduledCommand(ctx context.Context, id int64) (readd bool) {
	ctx, span := trace.StartSpan(ctx, "runScheduledCommand")
	defer span.End()

	runner := &scheduledCommandRunner{
		id:   id,
		deps: b.deps,
	}

	readd, err := b.runRepeat(ctx, runner)
	if err != nil {
		ctxlog.Warn(ctx, "error running scheduled command", zap.Error(err))
	} else {
		metricScheduled.Inc()
	}
	return readd
}

type repeatRunner interface {
	withLog(ctx context.Context) context.Context
	status(ctx context.Context, exec boil.ContextExecutor) (status repeatStatus, err error)
	load(ctx context.Context, exec boil.ContextExecutor) error
	channel() *models.Channel
	allowed(ctx context.Context) (found bool, allowed bool, err error)
	updateCount(ctx context.Context, exec boil.ContextExecutor) error
	info() *models.CommandInfo
}

type repeatStatus struct {
	Enabled bool `boil:"enabled"`
	Active  bool `boil:"active"`
	Ready   bool `boil:"ready"`
}

func (b *Bot) runRepeat(ctx context.Context, runner repeatRunner) (readd bool, err error) {
	readd = true

	ctx, span := trace.StartSpan(ctx, "runRepeat")
	defer span.End()

	ctx = runner.withLog(ctx)
	start := b.deps.Clock.Now()

	err = transact(ctx, b.db, func(ctx context.Context, tx *sql.Tx) error {
		status, err := runner.status(ctx, tx)
		if err != nil {
			if err == sql.ErrNoRows {
				status = repeatStatus{}
			} else {
				return err
			}
		}

		if !status.Enabled || !status.Active {
			readd = false
			return nil
		}

		if !status.Ready {
			return nil
		}

		if err := runner.load(ctx, tx); err != nil {
			if err == sql.ErrNoRows {
				readd = false
				return nil
			}
			return err
		}

		channel := runner.channel()
		// TODO: Remove if possible by passing the top level wqueue down here.
		if err := pgLock(ctx, tx, channel.UserID); err != nil {
			return err
		}

		found, allowed, err := runner.allowed(ctx)
		readd = readd && found
		if !allowed || err != nil {
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
		}

		info := runner.info()

		info.Count++

		if err := info.Update(ctx, s.Tx, boil.Whitelist(models.CommandInfoColumns.Count)); err != nil {
			return err
		}

		ctx = ctxlog.With(ctx, zap.Int64("roomID", s.RoomID), zap.String("channel", s.IRCChannel))

		var message string

		if command := info.R.CustomCommand; command != nil {
			message = command.Message
		} else {
			items := info.R.CommandList.Items

			if len(items) == 0 {
				return nil
			}

			i := s.Deps.Rand.Intn(len(items))
			message = items[i]
		}

		return runCommandAndCount(ctx, s, info, message, true)
	})

	return readd, err
}

type repeatedCommandRunner struct {
	id     int64
	deps   *sharedDeps
	repeat *models.RepeatedCommand
}

var _ repeatRunner = (*repeatedCommandRunner)(nil)

func (runner *repeatedCommandRunner) withLog(ctx context.Context) context.Context {
	trace.FromContext(ctx).AddAttributes(
		trace.Int64Attribute("repeatedCommand", runner.id),
	)
	return ctxlog.With(ctx, zap.Int64("repeatedCommand", runner.id))
}

func (runner *repeatedCommandRunner) status(ctx context.Context, exec boil.ContextExecutor) (status repeatStatus, err error) {
	ctx, span := trace.StartSpan(ctx, "repeatedCommandRunner.status")
	defer span.End()

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

func (runner *repeatedCommandRunner) allowed(ctx context.Context) (found bool, allowed bool, err error) {
	ctx, span := trace.StartSpan(ctx, "repeatedCommandRunner.allowed")
	defer span.End()

	channel := runner.channel()
	repeat := runner.repeat

	if !channel.Active || !repeat.Enabled {
		return false, false, nil
	}

	if channel.MessageCount < repeat.LastCount+repeat.MessageDiff {
		return true, false, nil
	}

	roomIDStr := strconv.FormatInt(channel.UserID, 10)
	expiry := time.Duration(repeat.Delay-1) * time.Second

	allowed, err = runner.deps.Redis.RepeatAllowed(ctx, roomIDStr, runner.id, expiry)
	return true, allowed, err
}

func (runner *repeatedCommandRunner) load(ctx context.Context, exec boil.ContextExecutor) error {
	ctx, span := trace.StartSpan(ctx, "repeatedCommandRunner.load")
	defer span.End()

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
	ctx, span := trace.StartSpan(ctx, "repeatedCommandRunner.updateCount")
	defer span.End()

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
	trace.FromContext(ctx).AddAttributes(
		trace.Int64Attribute("scheduledCommand", runner.id),
	)
	return ctxlog.With(ctx, zap.Int64("scheduledCommand", runner.id))
}

func (runner *scheduledCommandRunner) status(ctx context.Context, exec boil.ContextExecutor) (status repeatStatus, err error) {
	ctx, span := trace.StartSpan(ctx, "scheduledCommandRunner.status")
	defer span.End()

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

func (runner *scheduledCommandRunner) allowed(ctx context.Context) (found bool, allowed bool, err error) {
	ctx, span := trace.StartSpan(ctx, "scheduledCommandRunner.allowed")
	defer span.End()

	channel := runner.channel()
	scheduled := runner.scheduled

	if !channel.Active || !scheduled.Enabled {
		return false, false, nil
	}

	if channel.MessageCount < scheduled.LastCount+scheduled.MessageDiff {
		return true, false, nil
	}

	// Hardcoded to 29 seconds, since cron jobs run at a fixed schedule
	// according to the clock rather than at an interval with an arbitrary
	// offset. This prevents any given cron from running faster than every
	// 30 seconds.
	roomIDStr := strconv.FormatInt(channel.UserID, 10)
	allowed, err = runner.deps.Redis.ScheduledAllowed(ctx, roomIDStr, runner.id, 29*time.Second)
	return true, allowed, err
}

func (runner *scheduledCommandRunner) load(ctx context.Context, exec boil.ContextExecutor) error {
	ctx, span := trace.StartSpan(ctx, "scheduledCommandRunner.load")
	defer span.End()

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
	ctx, span := trace.StartSpan(ctx, "scheduledCommandRunner.updateCount")
	defer span.End()

	scheduled := runner.scheduled
	scheduled.LastCount = runner.channel().MessageCount
	return scheduled.Update(ctx, exec, boil.Whitelist(models.ScheduledCommandColumns.LastCount))
}

func (runner *scheduledCommandRunner) info() *models.CommandInfo {
	return runner.scheduled.R.CommandInfo
}

func (b *Bot) loadRepeats(ctx context.Context, reset bool) error {
	ctx, span := trace.StartSpan(ctx, "loadRepeats")
	defer span.End()

	if reset {
		if err := b.rep.Reset(ctx); err != nil {
			return err
		}
	}

	repeats, err := models.RepeatedCommands(
		models.RepeatedCommandWhere.Enabled.EQ(true),
	).All(ctx, b.db)
	if err != nil {
		return err
	}

	if err := updateRepeating(ctx, b.deps, repeats, true); err != nil {
		return err
	}

	scheduleds, err := models.ScheduledCommands(
		models.ScheduledCommandWhere.Enabled.EQ(true),
	).All(ctx, b.db)
	if err != nil {
		return err
	}

	return updateScheduleds(ctx, b.deps, scheduleds, true)
}

func updateRepeating(ctx context.Context, deps *sharedDeps, repeats []*models.RepeatedCommand, enable bool) error {
	for _, repeat := range repeats {
		if !enable || !repeat.Enabled {
			if err := deps.RemoveRepeat(ctx, repeat.ID); err != nil {
				return err
			}
			continue
		}

		interval := time.Duration(repeat.Delay) * time.Second

		start := repeat.UpdatedAt
		if repeat.InitTimestamp.Valid {
			start = repeat.InitTimestamp.Time
		}

		if err := deps.AddRepeat(ctx, repeat.ID, start, interval); err != nil {
			return err
		}
	}

	return nil
}

func updateScheduleds(ctx context.Context, deps *sharedDeps, scheduleds []*models.ScheduledCommand, enable bool) error {
	for _, scheduled := range scheduleds {
		if !enable || !scheduled.Enabled {
			if err := deps.RemoveScheduled(ctx, scheduled.ID); err != nil {
				return err
			}
			continue
		}

		expr, err := repeat.ParseCron(scheduled.CronExpression)
		if err != nil {
			panic(err)
		}

		if err := deps.AddScheduled(ctx, scheduled.ID, expr); err != nil {
			return err
		}
	}

	return nil
}
