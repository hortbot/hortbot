package botstate

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/hortbot/hortbot/internal/db/dbsql"
	"github.com/jackc/pgx/v5"
)

const (
	confirmationNamespace  = "confirmation"
	filterWarningNamespace = "filter-warning"
)

// LinkPermit grants a user a single link permit for the channel.
func (s *Store) LinkPermit(ctx context.Context, queries *dbsql.Queries, channel, user string, expiry time.Duration) error {
	now, err := s.currentTime(ctx, queries)
	if err != nil {
		return err
	}
	err = queries.BotStateGrantLinkPermit(ctx, dbsql.BotStateGrantLinkPermitParams{
		Channel:   channel,
		UserID:    user,
		ExpiresAt: dbsql.TimestamptzFrom(now.Add(expiry)),
	})
	if err != nil {
		return fmt.Errorf("link permit: %w", err)
	}
	return nil
}

// HasLinkPermit reports whether the user has an active link permit
// and consumes it.
func (s *Store) HasLinkPermit(ctx context.Context, queries *dbsql.Queries, channel, user string) (bool, error) {
	now, err := s.currentTime(ctx, queries)
	if err != nil {
		return false, err
	}
	n, err := queries.BotStateConsumeLinkPermit(ctx, dbsql.BotStateConsumeLinkPermitParams{
		Channel: channel,
		UserID:  user,
		Now:     dbsql.TimestamptzFrom(now),
	})
	if err != nil {
		return false, fmt.Errorf("has link permit: %w", err)
	}
	return n > 0, nil
}

// Confirm tracks a per-(channel, user, key) confirmation. Returns
// true the second time it is called within the expiry window and
// consumes the confirmation.
func (s *Store) Confirm(ctx context.Context, queries *dbsql.Queries, channel, user, key string, expiry time.Duration) (bool, error) {
	now, err := s.currentTime(ctx, queries)
	if err != nil {
		return false, err
	}
	lockKey := channel + "\x00" + user + "\x00" + key
	if err := withKeyLock(ctx, queries, confirmationNamespace, lockKey); err != nil {
		return false, err
	}

	current, err := queries.BotStateGetConfirmationExpiry(ctx, dbsql.BotStateGetConfirmationExpiryParams{
		Channel:         channel,
		UserID:          user,
		ConfirmationKey: key,
	})
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return false, fmt.Errorf("select confirmation: %w", err)
	}

	if err == nil && current.Time.After(now) {
		if err := queries.BotStateDeleteConfirmation(ctx, dbsql.BotStateDeleteConfirmationParams{
			Channel:         channel,
			UserID:          user,
			ConfirmationKey: key,
		}); err != nil {
			return false, fmt.Errorf("delete confirmation: %w", err)
		}
		return true, nil
	}

	if err := queries.BotStateUpsertConfirmation(ctx, dbsql.BotStateUpsertConfirmationParams{
		Channel:         channel,
		UserID:          user,
		ConfirmationKey: key,
		ExpiresAt:       dbsql.TimestamptzFrom(now.Add(expiry)),
	}); err != nil {
		return false, fmt.Errorf("upsert confirmation: %w", err)
	}
	return false, nil
}

// FilterWarned reports whether the user has already been warned about
// the given filter and refreshes the warning's expiry.
//
// The one-second initial expiry preserves the existing confirmation behavior.
// exec must be a transaction because the operation spans statements.
func (s *Store) FilterWarned(ctx context.Context, queries *dbsql.Queries, channel, user, filter string, expiry time.Duration) (bool, error) {
	now, err := s.currentTime(ctx, queries)
	if err != nil {
		return false, err
	}
	lockKey := channel + "\x00" + user + "\x00" + filter
	if err := withKeyLock(ctx, queries, filterWarningNamespace, lockKey); err != nil {
		return false, err
	}

	current, err := queries.BotStateGetFilterWarningExpiry(ctx, dbsql.BotStateGetFilterWarningExpiryParams{
		Channel:    channel,
		UserID:     user,
		FilterName: filter,
	})
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return false, fmt.Errorf("select filter warning: %w", err)
	}
	existed := err == nil && current.Time.After(now)

	expiresAt := now.Add(time.Second)
	if existed {
		expiresAt = now.Add(expiry)
	}
	if err := queries.BotStateUpsertFilterWarning(ctx, dbsql.BotStateUpsertFilterWarningParams{
		Channel:    channel,
		UserID:     user,
		FilterName: filter,
		ExpiresAt:  dbsql.TimestamptzFrom(expiresAt),
	}); err != nil {
		return false, fmt.Errorf("upsert filter warning: %w", err)
	}
	return existed, nil
}
