package botstate

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/hortbot/hortbot/internal/db/dbsql"
	"github.com/jackc/pgx/v5"
)

// MarkCooldown marks a command as on cooldown for the given window,
// replacing any existing entry.
func (s *Store) MarkCooldown(ctx context.Context, queries *dbsql.Queries, channel, command string, expiry time.Duration) error {
	now, err := s.currentTime(ctx, queries)
	if err != nil {
		return err
	}
	err = queries.BotStateMarkCommandCooldown(ctx, dbsql.BotStateMarkCommandCooldownParams{
		Channel:    channel,
		CommandKey: command,
		ExpiresAt:  dbsql.TimestamptzFrom(now.Add(expiry)),
	})
	if err != nil {
		return fmt.Errorf("mark command cooldown: %w", err)
	}
	return nil
}

// CheckAndMarkCooldown reports whether a command is currently on
// cooldown and, if not, starts the cooldown.
func (s *Store) CheckAndMarkCooldown(ctx context.Context, queries *dbsql.Queries, channel, command string, expiry time.Duration) (bool, error) {
	now, err := s.currentTime(ctx, queries)
	if err != nil {
		return false, err
	}
	marked, err := queries.BotStateCheckAndMarkCommandCooldown(ctx, dbsql.BotStateCheckAndMarkCommandCooldownParams{
		Channel:    channel,
		CommandKey: command,
		ExpiresAt:  dbsql.TimestamptzFrom(now.Add(expiry)),
		Now:        dbsql.TimestamptzFrom(now),
	})
	return scanCooldown(marked, err, "check and mark command cooldown")
}

// RepeatAllowed reports whether a repeated command may run, blocking
// further repeats with the same id until the expiry elapses.
func (s *Store) RepeatAllowed(ctx context.Context, queries *dbsql.Queries, channel string, id int64, expiry time.Duration) (bool, error) {
	now, err := s.currentTime(ctx, queries)
	if err != nil {
		return false, err
	}
	marked, err := queries.BotStateCheckAndMarkRepeatCooldown(ctx, dbsql.BotStateCheckAndMarkRepeatCooldownParams{
		Channel:           channel,
		RepeatedCommandID: id,
		ExpiresAt:         dbsql.TimestamptzFrom(now.Add(expiry)),
		Now:               dbsql.TimestamptzFrom(now),
	})
	seen, err := scanCooldown(marked, err, "check and mark repeat cooldown")
	return !seen, err
}

// ScheduledAllowed reports whether a scheduled command may run.
func (s *Store) ScheduledAllowed(ctx context.Context, queries *dbsql.Queries, channel string, id int64, expiry time.Duration) (bool, error) {
	now, err := s.currentTime(ctx, queries)
	if err != nil {
		return false, err
	}
	marked, err := queries.BotStateCheckAndMarkScheduledCooldown(ctx, dbsql.BotStateCheckAndMarkScheduledCooldownParams{
		Channel:            channel,
		ScheduledCommandID: id,
		ExpiresAt:          dbsql.TimestamptzFrom(now.Add(expiry)),
		Now:                dbsql.TimestamptzFrom(now),
	})
	seen, err := scanCooldown(marked, err, "check and mark scheduled command cooldown")
	return !seen, err
}

// AutoreplyAllowed reports whether an autoreply may run.
func (s *Store) AutoreplyAllowed(ctx context.Context, queries *dbsql.Queries, channel string, id int64, expiry time.Duration) (bool, error) {
	now, err := s.currentTime(ctx, queries)
	if err != nil {
		return false, err
	}
	marked, err := queries.BotStateCheckAndMarkAutoreplyCooldown(ctx, dbsql.BotStateCheckAndMarkAutoreplyCooldownParams{
		Channel:     channel,
		AutoreplyID: id,
		ExpiresAt:   dbsql.TimestamptzFrom(now.Add(expiry)),
		Now:         dbsql.TimestamptzFrom(now),
	})
	seen, err := scanCooldown(marked, err, "check and mark autoreply cooldown")
	return !seen, err
}

func scanCooldown(marked bool, err error, operation string) (bool, error) {
	switch {
	case err == nil:
		return false, nil
	case errors.Is(err, pgx.ErrNoRows):
		return true, nil
	default:
		return false, fmt.Errorf("%s: %w", operation, err)
	}
}
