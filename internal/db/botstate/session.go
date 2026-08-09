package botstate

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/aarondl/sqlboiler/v4/boil"
)

const (
	confirmationNamespace  = "confirmation"
	filterWarningNamespace = "filter-warning"
)

// LinkPermit grants a user a single link permit for the channel.
func (s *Store) LinkPermit(ctx context.Context, exec boil.ContextExecutor, channel, user string, expiry time.Duration) error {
	now, err := s.currentTime(ctx, exec)
	if err != nil {
		return err
	}
	_, err = exec.ExecContext(ctx, `
		INSERT INTO bot_link_permits (channel, user_id, expires_at)
		VALUES ($1, $2, $3)
		ON CONFLICT (channel, user_id) DO UPDATE SET expires_at = excluded.expires_at
	`, channel, user, now.Add(expiry))
	if err != nil {
		return fmt.Errorf("link permit: %w", err)
	}
	return nil
}

// HasLinkPermit reports whether the user has an active link permit
// and consumes it.
func (s *Store) HasLinkPermit(ctx context.Context, exec boil.ContextExecutor, channel, user string) (bool, error) {
	now, err := s.currentTime(ctx, exec)
	if err != nil {
		return false, err
	}
	res, err := exec.ExecContext(ctx, `
		DELETE FROM bot_link_permits
		WHERE channel = $1 AND user_id = $2 AND expires_at > $3
	`, channel, user, now)
	if err != nil {
		return false, fmt.Errorf("has link permit: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("rows affected: %w", err)
	}
	return n > 0, nil
}

// Confirm tracks a per-(channel, user, key) confirmation. Returns
// true the second time it is called within the expiry window and
// consumes the confirmation.
func (s *Store) Confirm(ctx context.Context, exec boil.ContextExecutor, channel, user, key string, expiry time.Duration) (bool, error) {
	now, err := s.currentTime(ctx, exec)
	if err != nil {
		return false, err
	}
	lockKey := channel + "\x00" + user + "\x00" + key
	if err := withKeyLock(ctx, exec, confirmationNamespace, lockKey); err != nil {
		return false, err
	}

	var current sql.NullTime
	err = exec.QueryRowContext(ctx, `
		SELECT expires_at FROM bot_confirmations
		WHERE channel = $1 AND user_id = $2 AND confirmation_key = $3
	`, channel, user, key).Scan(&current)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return false, fmt.Errorf("select confirmation: %w", err)
	}

	if current.Valid && current.Time.After(now) {
		if _, err := exec.ExecContext(ctx, `
			DELETE FROM bot_confirmations
			WHERE channel = $1 AND user_id = $2 AND confirmation_key = $3
		`, channel, user, key); err != nil {
			return false, fmt.Errorf("delete confirmation: %w", err)
		}
		return true, nil
	}

	if _, err := exec.ExecContext(ctx, `
		INSERT INTO bot_confirmations (channel, user_id, confirmation_key, expires_at)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (channel, user_id, confirmation_key) DO UPDATE
			SET expires_at = excluded.expires_at
	`, channel, user, key, now.Add(expiry)); err != nil {
		return false, fmt.Errorf("upsert confirmation: %w", err)
	}
	return false, nil
}

// FilterWarned reports whether the user has already been warned about
// the given filter and refreshes the warning's expiry.
//
// The one-second initial expiry preserves the existing Redis behavior.
// exec must be a transaction because the operation spans statements.
func (s *Store) FilterWarned(ctx context.Context, exec boil.ContextExecutor, channel, user, filter string, expiry time.Duration) (bool, error) {
	now, err := s.currentTime(ctx, exec)
	if err != nil {
		return false, err
	}
	lockKey := channel + "\x00" + user + "\x00" + filter
	if err := withKeyLock(ctx, exec, filterWarningNamespace, lockKey); err != nil {
		return false, err
	}

	var current sql.NullTime
	err = exec.QueryRowContext(ctx, `
		SELECT expires_at FROM bot_filter_warnings
		WHERE channel = $1 AND user_id = $2 AND filter_name = $3
	`, channel, user, filter).Scan(&current)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return false, fmt.Errorf("select filter warning: %w", err)
	}
	existed := current.Valid && current.Time.After(now)

	expiresAt := now.Add(time.Second)
	if existed {
		expiresAt = now.Add(expiry)
	}
	if _, err := exec.ExecContext(ctx, `
		INSERT INTO bot_filter_warnings (channel, user_id, filter_name, expires_at)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (channel, user_id, filter_name) DO UPDATE
			SET expires_at = excluded.expires_at
	`, channel, user, filter, expiresAt); err != nil {
		return false, fmt.Errorf("upsert filter warning: %w", err)
	}
	return existed, nil
}
