package botstate

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/hortbot/hortbot/internal/db/dbsql"
	"github.com/jackc/pgx/v5"
)

// SetAuthState stores an arbitrary JSON-encoded authentication state
// keyed by the given login-flow nonce.
func (s *Store) SetAuthState(ctx context.Context, queries *dbsql.Queries, key string, value any, expiry time.Duration) error {
	b, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("marshaling auth state: %w", err)
	}

	now, err := s.currentTime(ctx, queries)
	if err != nil {
		return err
	}
	if err := queries.BotStateSetAuthState(ctx, dbsql.BotStateSetAuthStateParams{
		Key:       key,
		Value:     b,
		ExpiresAt: dbsql.TimestamptzFrom(now.Add(expiry)),
	}); err != nil {
		return fmt.Errorf("set auth state: %w", err)
	}
	return nil
}

// GetAuthState retrieves and consumes a previously stored
// authentication state, decoding it into v. The state is removed
// regardless of whether unmarshalling succeeds.
//
// Implemented as a single atomic DELETE ... RETURNING value so two
// concurrent callers cannot both observe the same state.
func (s *Store) GetAuthState(ctx context.Context, queries *dbsql.Queries, key string, v any) (bool, error) {
	now, err := s.currentTime(ctx, queries)
	if err != nil {
		return false, err
	}

	raw, err := queries.BotStateTakeAuthState(ctx, dbsql.BotStateTakeAuthStateParams{
		Key: key,
		Now: dbsql.TimestamptzFrom(now),
	})
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return false, nil
	case err != nil:
		return false, fmt.Errorf("get auth state: %w", err)
	}

	if err := json.Unmarshal(raw, v); err != nil {
		return false, fmt.Errorf("unmarshaling auth state: %w", err)
	}
	return true, nil
}
