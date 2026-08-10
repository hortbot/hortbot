package botstate

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/hortbot/hortbot/internal/db/dbsql"
)

// Dump returns the contents of the TTL-bearing tables for debugging.
func (*Store) Dump(ctx context.Context, q *dbsql.Queries) (string, error) {
	var sb strings.Builder

	commandCooldowns, err := q.BotStateDumpCommandCooldowns(ctx)
	if err != nil {
		return "", fmt.Errorf("dump bot_command_cooldowns: %w", err)
	}
	appendDumpRows(&sb, "bot_command_cooldowns", commandCooldowns, func(row dbsql.BotStateDumpCommandCooldownsRow) (string, string, time.Time) {
		return row.Key, row.Value, row.ExpiresAt.Time
	})

	repeatCooldowns, err := q.BotStateDumpRepeatCooldowns(ctx)
	if err != nil {
		return "", fmt.Errorf("dump bot_repeat_cooldowns: %w", err)
	}
	appendDumpRows(&sb, "bot_repeat_cooldowns", repeatCooldowns, func(row dbsql.BotStateDumpRepeatCooldownsRow) (string, string, time.Time) {
		return row.Key, row.Value, row.ExpiresAt.Time
	})

	scheduledCooldowns, err := q.BotStateDumpScheduledCooldowns(ctx)
	if err != nil {
		return "", fmt.Errorf("dump bot_scheduled_command_cooldowns: %w", err)
	}
	appendDumpRows(&sb, "bot_scheduled_command_cooldowns", scheduledCooldowns, func(row dbsql.BotStateDumpScheduledCooldownsRow) (string, string, time.Time) {
		return row.Key, row.Value, row.ExpiresAt.Time
	})

	autoreplyCooldowns, err := q.BotStateDumpAutoreplyCooldowns(ctx)
	if err != nil {
		return "", fmt.Errorf("dump bot_autoreply_cooldowns: %w", err)
	}
	appendDumpRows(&sb, "bot_autoreply_cooldowns", autoreplyCooldowns, func(row dbsql.BotStateDumpAutoreplyCooldownsRow) (string, string, time.Time) {
		return row.Key, row.Value, row.ExpiresAt.Time
	})

	linkPermits, err := q.BotStateDumpLinkPermits(ctx)
	if err != nil {
		return "", fmt.Errorf("dump bot_link_permits: %w", err)
	}
	appendDumpRows(&sb, "bot_link_permits", linkPermits, func(row dbsql.BotStateDumpLinkPermitsRow) (string, string, time.Time) {
		return row.Key, row.Value, row.ExpiresAt.Time
	})

	confirmations, err := q.BotStateDumpConfirmations(ctx)
	if err != nil {
		return "", fmt.Errorf("dump bot_confirmations: %w", err)
	}
	appendDumpRows(&sb, "bot_confirmations", confirmations, func(row dbsql.BotStateDumpConfirmationsRow) (string, string, time.Time) {
		return row.Key, row.Value, row.ExpiresAt.Time
	})

	filterWarnings, err := q.BotStateDumpFilterWarnings(ctx)
	if err != nil {
		return "", fmt.Errorf("dump bot_filter_warnings: %w", err)
	}
	appendDumpRows(&sb, "bot_filter_warnings", filterWarnings, func(row dbsql.BotStateDumpFilterWarningsRow) (string, string, time.Time) {
		return row.Key, row.Value, row.ExpiresAt.Time
	})

	authStates, err := q.BotStateDumpAuthStates(ctx)
	if err != nil {
		return "", fmt.Errorf("dump web_auth_states: %w", err)
	}
	appendDumpRows(&sb, "web_auth_states", authStates, func(row dbsql.BotStateDumpAuthStatesRow) (string, string, time.Time) {
		return row.Key, row.Value, row.ExpiresAt.Time
	})

	return sb.String(), nil
}

func appendDumpRows[T any](sb *strings.Builder, table string, rows []T, fields func(T) (string, string, time.Time)) {
	for _, row := range rows {
		key, value, expiresAt := fields(row)
		fmt.Fprintf(sb, "%s\t%s\t%s\t%s\n", table, key, value, expiresAt.Format(time.RFC3339Nano))
	}
}
