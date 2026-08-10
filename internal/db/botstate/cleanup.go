package botstate

import (
	"context"
	"fmt"

	"github.com/hortbot/hortbot/internal/db/dbsql"
)

// Cleanup deletes expired rows using PostgreSQL's clock.
func (*Store) Cleanup(ctx context.Context, q *dbsql.Queries) error {
	for _, cleanup := range []func(context.Context) error{
		q.BotStateCleanupCommandCooldowns,
		q.BotStateCleanupRepeatCooldowns,
		q.BotStateCleanupScheduledCooldowns,
		q.BotStateCleanupAutoreplyCooldowns,
		q.BotStateCleanupLinkPermits,
		q.BotStateCleanupConfirmations,
		q.BotStateCleanupFilterWarnings,
		q.BotStateCleanupAuthStates,
	} {
		if err := cleanup(ctx); err != nil {
			return fmt.Errorf("delete expired rows: %w", err)
		}
	}
	return nil
}
