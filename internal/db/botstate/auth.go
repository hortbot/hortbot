package botstate

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/aarondl/sqlboiler/v4/boil"
)

// SetAuthState stores an arbitrary JSON-encoded authentication state
// keyed by the given login-flow nonce.
func (s *Store) SetAuthState(ctx context.Context, exec boil.ContextExecutor, key string, value any, expiry time.Duration) error {
	b, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("marshaling auth state: %w", err)
	}

	now, err := s.currentTime(ctx, exec)
	if err != nil {
		return err
	}
	if _, err := exec.ExecContext(ctx, `
		INSERT INTO web_auth_states (key, value, expires_at) VALUES ($1, $2, $3)
		ON CONFLICT (key) DO UPDATE
			SET value = excluded.value,
			    expires_at = excluded.expires_at
	`, key, b, now.Add(expiry)); err != nil {
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
func (s *Store) GetAuthState(ctx context.Context, exec boil.ContextExecutor, key string, v any) (bool, error) {
	now, err := s.currentTime(ctx, exec)
	if err != nil {
		return false, err
	}

	var raw []byte
	err = exec.QueryRowContext(ctx, `
		DELETE FROM web_auth_states WHERE key = $1 AND expires_at > $2 RETURNING value
	`, key, now).Scan(&raw)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return false, nil
	case err != nil:
		return false, fmt.Errorf("get auth state: %w", err)
	}

	if err := json.Unmarshal(raw, v); err != nil {
		return false, fmt.Errorf("unmarshaling auth state: %w", err)
	}
	return true, nil
}
