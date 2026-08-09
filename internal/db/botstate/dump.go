package botstate

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/aarondl/sqlboiler/v4/boil"
)

// Dump returns the contents of the TTL-bearing tables for debugging.
func (*Store) Dump(ctx context.Context, exec boil.ContextExecutor) (string, error) {
	var sb strings.Builder

	queries := []struct {
		table string
		query string
	}{
		{"bot_command_cooldowns", `
			SELECT channel || '/' || command_key, ''::text, expires_at
			FROM bot_command_cooldowns ORDER BY channel, command_key
		`},
		{"bot_repeat_cooldowns", `
			SELECT channel || '/' || repeated_command_id, ''::text, expires_at
			FROM bot_repeat_cooldowns ORDER BY channel, repeated_command_id
		`},
		{"bot_scheduled_command_cooldowns", `
			SELECT channel || '/' || scheduled_command_id, ''::text, expires_at
			FROM bot_scheduled_command_cooldowns ORDER BY channel, scheduled_command_id
		`},
		{"bot_autoreply_cooldowns", `
			SELECT channel || '/' || autoreply_id, ''::text, expires_at
			FROM bot_autoreply_cooldowns ORDER BY channel, autoreply_id
		`},
		{"bot_link_permits", `
			SELECT channel || '/' || user_id, ''::text, expires_at
			FROM bot_link_permits ORDER BY channel, user_id
		`},
		{"bot_confirmations", `
			SELECT channel || '/' || user_id || '/' || confirmation_key, ''::text, expires_at
			FROM bot_confirmations ORDER BY channel, user_id, confirmation_key
		`},
		{"bot_filter_warnings", `
			SELECT channel || '/' || user_id || '/' || filter_name, ''::text, expires_at
			FROM bot_filter_warnings ORDER BY channel, user_id, filter_name
		`},
		{"web_auth_states", `
			SELECT key, encode(value, 'escape'), expires_at
			FROM web_auth_states ORDER BY key
		`},
	}

	for _, q := range queries {
		if err := dumpQuery(ctx, exec, &sb, q.table, q.query); err != nil {
			return "", err
		}
	}
	return sb.String(), nil
}

func dumpQuery(ctx context.Context, exec boil.ContextExecutor, sb *strings.Builder, table, query string) error {
	rows, err := exec.QueryContext(ctx, query)
	if err != nil {
		return fmt.Errorf("dump %s: %w", table, err)
	}

	for rows.Next() {
		var key, value string
		var expiresAt time.Time
		if err := rows.Scan(&key, &value, &expiresAt); err != nil {
			_ = rows.Close()
			return fmt.Errorf("dump %s row: %w", table, err)
		}
		fmt.Fprintf(sb, "%s\t%s\t%s\t%s\n", table, key, value, expiresAt.Format(time.RFC3339Nano))
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return fmt.Errorf("dump %s rows: %w", table, err)
	}
	if err := rows.Close(); err != nil {
		return fmt.Errorf("close %s dump: %w", table, err)
	}
	return nil
}
