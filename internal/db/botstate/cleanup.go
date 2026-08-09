package botstate

import (
	"context"
	"fmt"

	"github.com/aarondl/sqlboiler/v4/boil"
)

// Cleanup deletes expired rows using PostgreSQL's clock.
func (*Store) Cleanup(ctx context.Context, exec boil.ContextExecutor) error {
	for _, q := range []string{
		`DELETE FROM bot_command_cooldowns           WHERE expires_at < now()`,
		`DELETE FROM bot_repeat_cooldowns            WHERE expires_at < now()`,
		`DELETE FROM bot_scheduled_command_cooldowns WHERE expires_at < now()`,
		`DELETE FROM bot_autoreply_cooldowns         WHERE expires_at < now()`,
		`DELETE FROM bot_link_permits                WHERE expires_at < now()`,
		`DELETE FROM bot_confirmations               WHERE expires_at < now()`,
		`DELETE FROM bot_filter_warnings             WHERE expires_at < now()`,
		`DELETE FROM web_auth_states                 WHERE expires_at < now()`,
	} {
		if _, err := exec.ExecContext(ctx, q); err != nil {
			return fmt.Errorf("delete expired rows: %w", err)
		}
	}
	return nil
}
