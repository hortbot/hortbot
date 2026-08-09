// Package eventsubsync coordinates EventSub subscription synchronization through PostgreSQL.
package eventsubsync

import (
	"context"
	"fmt"

	"github.com/aarondl/sqlboiler/v4/boil"
)

type Requests struct{}

func (Requests) NotifyEventsubUpdates(ctx context.Context, exec boil.ContextExecutor) error {
	_, err := exec.ExecContext(ctx, `
		UPDATE eventsub_sync_requests
		SET version = version + 1
		WHERE singleton
	`)
	if err != nil {
		return fmt.Errorf("request eventsub sync: %w", err)
	}
	return nil
}

func (Requests) Version(ctx context.Context, exec boil.ContextExecutor) (int64, error) {
	var version int64
	err := exec.QueryRowContext(ctx, `
		SELECT version
		FROM eventsub_sync_requests
		WHERE singleton
	`).Scan(&version)
	if err != nil {
		return 0, fmt.Errorf("get eventsub sync version: %w", err)
	}
	return version, nil
}
