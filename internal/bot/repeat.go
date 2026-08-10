package bot

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/hortbot/hortbot/internal/db/dbsql"
	"github.com/hortbot/hortbot/internal/pkg/dbx"
	"github.com/hortbot/hortbot/internal/pkg/must"
	"github.com/hortbot/hortbot/internal/pkg/repeat"
	"github.com/jackc/pgx/v5"
	"github.com/zikaeroh/ctxlog"
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
	runner := &repeatedCommandRunner{
		id:   id,
		deps: b.deps,
	}

	readd, err := b.runRepeat(ctx, runner)
	if err != nil {
		metricRepeatedError.Inc()
		ctxlog.Warn(ctx, "error running repeated command", zap.Error(err), zap.Int64("id", id))
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
	runner := &scheduledCommandRunner{
		id:   id,
		deps: b.deps,
	}

	readd, err := b.runRepeat(ctx, runner)
	if err != nil {
		metricScheduledError.Inc()
		ctxlog.Warn(ctx, "error running scheduled command", zap.Error(err), zap.Int64("id", id))
	} else {
		metricScheduled.Inc()
	}
	return readd
}

type repeatRunner interface {
	withLog(ctx context.Context) context.Context
	status(ctx context.Context, queries *dbsql.Queries) (status repeatStatus, err error)
	load(ctx context.Context, queries *dbsql.Queries) error
	channel() *dbsql.Channel
	allowed(ctx context.Context, queries *dbsql.Queries) (found bool, allowed bool, err error)
	updateCount(ctx context.Context, queries *dbsql.Queries) error
	info() *dbsql.CommandInfo
	command() (string, []string)
}

type repeatStatus struct {
	Enabled bool
	Active  bool
	Ready   bool
}

func (b *Bot) runRepeat(ctx context.Context, runner repeatRunner) (readd bool, err error) {
	readd = true

	defer setMetricRepeatGauges(ctx, b.rep)

	ctx = runner.withLog(ctx)
	start := time.Now()

	err = dbx.Transact(ctx, b.db,
		dbx.SetLocalLockTimeout(5*time.Second),
		func(ctx context.Context, tx pgx.Tx) error {
			queries := dbsql.New(tx)

			status, err := runner.status(ctx, queries)
			if err != nil {
				if errors.Is(err, pgx.ErrNoRows) {
					status = repeatStatus{}
				} else {
					return fmt.Errorf("getting status: %w", err)
				}
			}

			if !status.Enabled || !status.Active {
				readd = false
				return nil
			}

			if !status.Ready {
				return nil
			}

			if err := runner.load(ctx, queries); err != nil {
				if errors.Is(err, pgx.ErrNoRows) {
					readd = false
					return nil
				}
				return fmt.Errorf("loading repeat: %w", err)
			}

			channel := runner.channel()
			// Serialize chat messages and repeat jobs for each channel.
			if err := pgLock(ctx, queries, channel.TwitchID); err != nil {
				return err
			}

			found, allowed, err := runner.allowed(ctx, queries)
			readd = readd && found
			if !allowed || err != nil {
				return err //nolint:wrapcheck
			}

			if err := runner.updateCount(ctx, queries); err != nil {
				return fmt.Errorf("updating count: %w", err)
			}

			s := &session{
				Type:        sessionRepeat,
				Deps:        b.deps,
				Queries:     queries,
				Start:       start,
				UserLevel:   AccessLevelEveryone,
				Channel:     channel,
				BotLogin:    channel.BotName,
				ChannelName: channel.Name,
				RoomID:      channel.TwitchID,
				RoomIDOrig:  channel.TwitchID,
			}

			info := runner.info()

			info.Count++

			if err := s.Queries.UpdateCommandInfoCount(ctx, dbsql.UpdateCommandInfoCountParams{
				Count: info.Count,
				ID:    info.ID,
			}); err != nil {
				return fmt.Errorf("updating command info: %w", err)
			}

			ctx = ctxlog.With(ctx, zap.Int64("roomID", s.RoomID), zap.String("channel", s.ChannelName))

			message, items := runner.command()
			if len(items) != 0 {
				message = items[s.Deps.Rand.Intn(len(items))]
			} else if message == "" {
				return nil
			}

			return runCommandAndCount(ctx, s, info, message, true)
		})

	return readd, err
}

type repeatedCommandRunner struct {
	id            int64
	deps          *sharedDeps
	repeat        *dbsql.RepeatedCommand
	loadedChannel *dbsql.Channel
	loadedInfo    *dbsql.CommandInfo
	message       string
	items         []string
}

var _ repeatRunner = (*repeatedCommandRunner)(nil)

func (runner *repeatedCommandRunner) withLog(ctx context.Context) context.Context {
	return ctxlog.With(ctx, zap.Int64("repeatedCommand", runner.id))
}

func (runner *repeatedCommandRunner) status(ctx context.Context, queries *dbsql.Queries) (status repeatStatus, err error) {
	row, err := queries.GetRepeatedCommandStatus(ctx, runner.id)
	if err != nil {
		return repeatStatus{}, fmt.Errorf("getting status: %w", err)
	}
	return repeatStatus(row), nil
}

func (runner *repeatedCommandRunner) allowed(ctx context.Context, queries *dbsql.Queries) (found bool, allowed bool, err error) {
	channel := runner.channel()
	repeat := runner.repeat

	if !channel.Active || !repeat.Enabled {
		return false, false, nil
	}

	if channel.MessageCount < repeat.LastCount+repeat.MessageDiff {
		return true, false, nil
	}

	roomIDStr := strconv.FormatInt(channel.TwitchID, 10)
	expiry := time.Duration(repeat.Delay-1) * time.Second

	allowed, err = runner.deps.State.RepeatAllowed(ctx, queries, roomIDStr, runner.id, expiry)
	if err != nil {
		return true, false, fmt.Errorf("checking if allowed: %w", err)
	}
	return true, allowed, nil
}

func (runner *repeatedCommandRunner) load(ctx context.Context, queries *dbsql.Queries) error {
	row, err := queries.GetRepeatedCommandForRun(ctx, runner.id)
	if err != nil {
		return fmt.Errorf("loading repeat: %w", err)
	}
	runner.repeat = new(row)
	return runner.loadRelations(ctx, queries)
}

func (runner *repeatedCommandRunner) channel() *dbsql.Channel {
	return runner.loadedChannel
}

func (runner *repeatedCommandRunner) updateCount(ctx context.Context, queries *dbsql.Queries) error {
	repeat := runner.repeat
	repeat.LastCount = runner.channel().MessageCount

	if err := queries.UpdateRepeatedCommandLastCount(ctx, dbsql.UpdateRepeatedCommandLastCountParams{
		LastCount: repeat.LastCount,
		ID:        repeat.ID,
	}); err != nil {
		return fmt.Errorf("updating count: %w", err)
	}
	return nil
}

func (runner *repeatedCommandRunner) info() *dbsql.CommandInfo {
	return runner.loadedInfo
}

func (runner *repeatedCommandRunner) command() (string, []string) {
	return runner.message, runner.items
}

func (runner *repeatedCommandRunner) loadRelations(ctx context.Context, queries *dbsql.Queries) error {
	channelRow, err := queries.GetChannelByID(ctx, runner.repeat.ChannelID)
	if err != nil {
		return fmt.Errorf("loading repeat channel: %w", err)
	}
	info, err := queries.GetCommandInfoByIDForUpdate(ctx, runner.repeat.CommandInfoID)
	if err != nil {
		return fmt.Errorf("loading repeat command info: %w", err)
	}
	runner.loadedChannel = &channelRow
	runner.loadedInfo = &info
	if info.CustomCommandID.Valid {
		command, err := queries.GetCustomCommand(ctx, info.CustomCommandID.Int64)
		if err != nil {
			return fmt.Errorf("loading repeat command: %w", err)
		}
		runner.message = command.Message
	} else {
		list, err := queries.GetCommandList(ctx, info.CommandListID.Int64)
		if err != nil {
			return fmt.Errorf("loading repeat list: %w", err)
		}
		runner.items = list.Items
	}
	return nil
}

type scheduledCommandRunner struct {
	id            int64
	deps          *sharedDeps
	scheduled     *dbsql.ScheduledCommand
	loadedChannel *dbsql.Channel
	loadedInfo    *dbsql.CommandInfo
	message       string
	items         []string
}

var _ repeatRunner = (*scheduledCommandRunner)(nil)

func (runner *scheduledCommandRunner) withLog(ctx context.Context) context.Context {
	return ctxlog.With(ctx, zap.Int64("scheduledCommand", runner.id))
}

func (runner *scheduledCommandRunner) status(ctx context.Context, queries *dbsql.Queries) (status repeatStatus, err error) {
	row, err := queries.GetScheduledCommandStatus(ctx, runner.id)
	if err != nil {
		return repeatStatus{}, fmt.Errorf("getting status: %w", err)
	}
	return repeatStatus(row), nil
}

func (runner *scheduledCommandRunner) allowed(ctx context.Context, queries *dbsql.Queries) (found bool, allowed bool, err error) {
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
	roomIDStr := strconv.FormatInt(channel.TwitchID, 10)
	allowed, err = runner.deps.State.ScheduledAllowed(ctx, queries, roomIDStr, runner.id, 29*time.Second)
	if err != nil {
		return true, false, fmt.Errorf("checking if allowed: %w", err)
	}
	return true, allowed, nil
}

func (runner *scheduledCommandRunner) load(ctx context.Context, queries *dbsql.Queries) error {
	row, err := queries.GetScheduledCommandForRun(ctx, runner.id)
	if err != nil {
		return fmt.Errorf("loading scheduled: %w", err)
	}
	runner.scheduled = new(row)
	return runner.loadRelations(ctx, queries)
}

func (runner *scheduledCommandRunner) channel() *dbsql.Channel {
	return runner.loadedChannel
}

func (runner *scheduledCommandRunner) updateCount(ctx context.Context, queries *dbsql.Queries) error {
	scheduled := runner.scheduled
	scheduled.LastCount = runner.channel().MessageCount

	if err := queries.UpdateScheduledCommandLastCount(ctx, dbsql.UpdateScheduledCommandLastCountParams{
		LastCount: scheduled.LastCount,
		ID:        scheduled.ID,
	}); err != nil {
		return fmt.Errorf("updating count: %w", err)
	}
	return nil
}

func (runner *scheduledCommandRunner) info() *dbsql.CommandInfo {
	return runner.loadedInfo
}

func (runner *scheduledCommandRunner) command() (string, []string) {
	return runner.message, runner.items
}

func (runner *scheduledCommandRunner) loadRelations(ctx context.Context, queries *dbsql.Queries) error {
	channelRow, err := queries.GetChannelByID(ctx, runner.scheduled.ChannelID)
	if err != nil {
		return fmt.Errorf("loading scheduled channel: %w", err)
	}
	info, err := queries.GetCommandInfoByIDForUpdate(ctx, runner.scheduled.CommandInfoID)
	if err != nil {
		return fmt.Errorf("loading scheduled command info: %w", err)
	}
	runner.loadedChannel = &channelRow
	runner.loadedInfo = &info
	if info.CustomCommandID.Valid {
		command, err := queries.GetCustomCommand(ctx, info.CustomCommandID.Int64)
		if err != nil {
			return fmt.Errorf("loading scheduled command: %w", err)
		}
		runner.message = command.Message
	} else {
		list, err := queries.GetCommandList(ctx, info.CommandListID.Int64)
		if err != nil {
			return fmt.Errorf("loading scheduled list: %w", err)
		}
		runner.items = list.Items
	}
	return nil
}

func (b *Bot) loadRepeats(ctx context.Context) error {
	defer setMetricRepeatGauges(ctx, b.rep)

	if err := b.rep.Reset(ctx); err != nil {
		return err
	}

	repeats, err := b.queries.ListActiveRepeatedCommands(ctx)
	if err != nil {
		return fmt.Errorf("getting active repeated commands: %w", err)
	}

	if err := updateRepeating(ctx, b.deps, repeats, true); err != nil {
		return err
	}

	scheduleds, err := b.queries.ListActiveScheduledCommands(ctx)
	if err != nil {
		return fmt.Errorf("getting active scheduled commands: %w", err)
	}

	return updateScheduleds(ctx, b.deps, scheduleds, true)
}

func updateRepeating(ctx context.Context, deps *sharedDeps, repeats []dbsql.RepeatedCommand, enable bool) error {
	for _, repeat := range repeats {
		if !enable || !repeat.Enabled {
			if err := deps.RemoveRepeat(ctx, repeat.ID); err != nil {
				return err
			}
			continue
		}

		interval := time.Duration(repeat.Delay) * time.Second

		start := repeat.UpdatedAt.Time
		if repeat.InitTimestamp.Valid {
			start = repeat.InitTimestamp.Time
		}

		if err := deps.AddRepeat(ctx, repeat.ID, start, interval); err != nil {
			return err
		}
	}

	return nil
}

func updateScheduleds(ctx context.Context, deps *sharedDeps, scheduleds []dbsql.ScheduledCommand, enable bool) error {
	for _, scheduled := range scheduleds {
		if !enable || !scheduled.Enabled {
			if err := deps.RemoveScheduled(ctx, scheduled.ID); err != nil {
				return err
			}
			continue
		}

		expr := must.Must(repeat.ParseCron(scheduled.CronExpression))

		if err := deps.AddScheduled(ctx, scheduled.ID, expr); err != nil {
			return err
		}
	}

	return nil
}
