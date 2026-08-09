package botstate

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/aarondl/sqlboiler/v4/boil"
)

// MarkCooldown marks a command as on cooldown for the given window,
// replacing any existing entry.
func (s *Store) MarkCooldown(ctx context.Context, exec boil.ContextExecutor, channel, command string, expiry time.Duration) error {
	now, err := s.currentTime(ctx, exec)
	if err != nil {
		return err
	}
	_, err = exec.ExecContext(ctx, `
		INSERT INTO bot_command_cooldowns (channel, command_key, expires_at) VALUES ($1, $2, $3)
		ON CONFLICT (channel, command_key) DO UPDATE SET expires_at = excluded.expires_at
	`, channel, command, now.Add(expiry))
	if err != nil {
		return fmt.Errorf("mark command cooldown: %w", err)
	}
	return nil
}

// CheckAndMarkCooldown reports whether a command is currently on
// cooldown and, if not, starts the cooldown.
func (s *Store) CheckAndMarkCooldown(ctx context.Context, exec boil.ContextExecutor, channel, command string, expiry time.Duration) (bool, error) {
	now, err := s.currentTime(ctx, exec)
	if err != nil {
		return false, err
	}
	row := exec.QueryRowContext(ctx, `
		INSERT INTO bot_command_cooldowns (channel, command_key, expires_at) VALUES ($1, $2, $3)
		ON CONFLICT (channel, command_key) DO UPDATE
			SET expires_at = excluded.expires_at
			WHERE bot_command_cooldowns.expires_at <= $4
		RETURNING true
	`, channel, command, now.Add(expiry), now)
	return scanCooldown(row, "check and mark command cooldown")
}

// RepeatAllowed reports whether a repeated command may run, blocking
// further repeats with the same id until the expiry elapses.
func (s *Store) RepeatAllowed(ctx context.Context, exec boil.ContextExecutor, channel string, id int64, expiry time.Duration) (bool, error) {
	now, err := s.currentTime(ctx, exec)
	if err != nil {
		return false, err
	}
	row := exec.QueryRowContext(ctx, `
		INSERT INTO bot_repeat_cooldowns (channel, repeated_command_id, expires_at) VALUES ($1, $2, $3)
		ON CONFLICT (channel, repeated_command_id) DO UPDATE
			SET expires_at = excluded.expires_at
			WHERE bot_repeat_cooldowns.expires_at <= $4
		RETURNING true
	`, channel, id, now.Add(expiry), now)
	seen, err := scanCooldown(row, "check and mark repeat cooldown")
	return !seen, err
}

// ScheduledAllowed reports whether a scheduled command may run.
func (s *Store) ScheduledAllowed(ctx context.Context, exec boil.ContextExecutor, channel string, id int64, expiry time.Duration) (bool, error) {
	now, err := s.currentTime(ctx, exec)
	if err != nil {
		return false, err
	}
	row := exec.QueryRowContext(ctx, `
		INSERT INTO bot_scheduled_command_cooldowns (channel, scheduled_command_id, expires_at) VALUES ($1, $2, $3)
		ON CONFLICT (channel, scheduled_command_id) DO UPDATE
			SET expires_at = excluded.expires_at
			WHERE bot_scheduled_command_cooldowns.expires_at <= $4
		RETURNING true
	`, channel, id, now.Add(expiry), now)
	seen, err := scanCooldown(row, "check and mark scheduled command cooldown")
	return !seen, err
}

// AutoreplyAllowed reports whether an autoreply may run.
func (s *Store) AutoreplyAllowed(ctx context.Context, exec boil.ContextExecutor, channel string, id int64, expiry time.Duration) (bool, error) {
	now, err := s.currentTime(ctx, exec)
	if err != nil {
		return false, err
	}
	row := exec.QueryRowContext(ctx, `
		INSERT INTO bot_autoreply_cooldowns (channel, autoreply_id, expires_at) VALUES ($1, $2, $3)
		ON CONFLICT (channel, autoreply_id) DO UPDATE
			SET expires_at = excluded.expires_at
			WHERE bot_autoreply_cooldowns.expires_at <= $4
		RETURNING true
	`, channel, id, now.Add(expiry), now)
	seen, err := scanCooldown(row, "check and mark autoreply cooldown")
	return !seen, err
}

func scanCooldown(row *sql.Row, operation string) (bool, error) {
	var marked bool
	switch err := row.Scan(&marked); {
	case err == nil:
		return false, nil
	case errors.Is(err, sql.ErrNoRows):
		return true, nil
	default:
		return false, fmt.Errorf("%s: %w", operation, err)
	}
}
