// Package eventsubsync coordinates EventSub subscription synchronization through PostgreSQL.
package eventsubsync

import (
	"context"
	"fmt"

	"github.com/hortbot/hortbot/internal/db/dbsql"
)

type Requests struct{}

func (Requests) NotifyEventsubUpdates(ctx context.Context, queries *dbsql.Queries) error {
	err := queries.RequestEventsubSync(ctx)
	if err != nil {
		return fmt.Errorf("request eventsub sync: %w", err)
	}
	return nil
}

func (Requests) Version(ctx context.Context, queries *dbsql.Queries) (int64, error) {
	version, err := queries.GetEventsubSyncVersion(ctx)
	if err != nil {
		return 0, fmt.Errorf("get eventsub sync version: %w", err)
	}
	return version, nil
}
