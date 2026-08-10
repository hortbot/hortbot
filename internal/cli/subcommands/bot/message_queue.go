package bot

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	corebot "github.com/hortbot/hortbot/internal/bot"
	"github.com/hortbot/hortbot/internal/bot/eventsubtobot"
	"github.com/hortbot/hortbot/internal/db/botstate"
	"github.com/hortbot/hortbot/internal/db/chatqueue"
	"github.com/hortbot/hortbot/internal/db/dbsql"
	"github.com/hortbot/hortbot/internal/pkg/apiclient/twitch/eventsub"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/zikaeroh/ctxlog"
	"go.uber.org/zap"
)

const (
	messageQueuePollInterval    = time.Second
	databaseMaintenanceInterval = time.Minute
	messageQueueFinalizeTimeout = 10 * time.Second
	messageQueueRetryDelay      = time.Second
)

type botLoginMapGetter func(context.Context) (map[int64]string, error)

func runQueueListener(ctx context.Context, queue *chatqueue.Queue, connString string) error {
	return retryQueueOperation(ctx, "listen for chat messages", func() error {
		return queue.Listen(ctx, connString)
	})
}

func runMessageWorker(
	ctx context.Context,
	workCtx context.Context,
	queue *chatqueue.Queue,
	b *corebot.Bot,
	maxAge time.Duration,
	getBotLoginMap botLoginMapGetter,
	poll <-chan time.Time,
) error {
	leaseDuration := max(30*time.Second, maxAge+15*time.Second)

	for {
		var lease *chatqueue.Lease
		err := retryQueueOperation(ctx, "claim chat message", func() error {
			var err error
			lease, err = queue.Claim(ctx, leaseDuration)
			if err != nil {
				return fmt.Errorf("claim chat message: %w", err)
			}
			return nil
		})
		if err != nil {
			return fmt.Errorf("claim queued message: %w", err)
		}
		if lease == nil {
			select {
			case <-queue.Wake():
			case <-poll:
			case <-ctx.Done():
				return ctx.Err()
			}
			continue
		}

		if time.Since(lease.MessageTimestamp) > maxAge {
			err := finishQueueOperation(workCtx, "complete stale chat message", func(ctx context.Context) error {
				return queue.Complete(ctx, lease)
			})
			if errors.Is(err, chatqueue.ErrLeaseLost) {
				continue
			}
			if err != nil {
				return fmt.Errorf("complete stale queued message: %w", err)
			}
			metricMessagesDropped.WithLabelValues("stale").Inc()
			continue
		}

		var raw eventsub.WebsocketMessage
		if err := json.Unmarshal(lease.Payload, &raw); err != nil {
			failErr := finishQueueOperation(workCtx, "fail invalid chat message", func(ctx context.Context) error {
				return queue.Fail(ctx, lease, err)
			})
			if errors.Is(failErr, chatqueue.ErrLeaseLost) {
				continue
			}
			if failErr != nil {
				return fmt.Errorf("fail invalid queued message: %w", failErr)
			}
			metricMessagesDropped.WithLabelValues("invalid").Inc()
			ctxlog.Error(ctx, "invalid queued message", zap.String("message_id", lease.ID), zap.Error(err))
			continue
		}

		botLoginMap, err := getBotLoginMap(workCtx)
		if err != nil {
			return err
		}
		message := eventsubtobot.ToMessage(botLoginMap, &raw)
		if err := b.HandleQueued(workCtx, message, lease.EnqueuedAt); err != nil {
			failErr := finishQueueOperation(workCtx, "fail chat message", func(ctx context.Context) error {
				return queue.Fail(ctx, lease, err)
			})
			if errors.Is(failErr, chatqueue.ErrLeaseLost) {
				continue
			}
			if failErr != nil {
				return fmt.Errorf("fail queued message: %w", failErr)
			}
			metricMessagesDropped.WithLabelValues("permanent").Inc()
			ctxlog.Error(ctx, "queued message failed without retry",
				zap.String("message_id", lease.ID),
				zap.Error(err),
			)
			continue
		}
		err = completeHandledMessage(workCtx, queue, lease)
		if errors.Is(err, chatqueue.ErrLeaseLost) {
			ctxlog.Error(ctx, "handled message lease was lost before completion",
				zap.String("message_id", lease.ID),
			)
			continue
		}
		if err != nil {
			return fmt.Errorf("complete queued message: %w", err)
		}
	}
}

func completeHandledMessage(ctx context.Context, queue *chatqueue.Queue, lease *chatqueue.Lease) error {
	return finishQueueOperation(ctx, "complete chat message", func(ctx context.Context) error {
		return queue.Complete(ctx, lease)
	})
}

func finishQueueOperation(ctx context.Context, operation string, fn func(context.Context) error) error {
	finishCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), messageQueueFinalizeTimeout)
	defer cancel()

	return retryQueueOperation(finishCtx, operation, func() error {
		return fn(finishCtx)
	})
}

func runDatabaseMaintenance(
	ctx context.Context,
	db *pgxpool.Pool,
	state *botstate.Store,
) error {
	ticker := time.NewTicker(databaseMaintenanceInterval)
	defer ticker.Stop()
	queries := dbsql.New(db)

	for {
		if err := state.Cleanup(ctx, queries); err != nil {
			ctxlog.Error(ctx, "error cleaning up bot state", zap.Error(err))
		}

		select {
		case <-ticker.C:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func retryQueueOperation(ctx context.Context, operation string, fn func() error) error {
	for {
		err := fn()
		if err == nil || errors.Is(err, chatqueue.ErrLeaseLost) {
			return err
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}
		ctxlog.Warn(ctx, "queue operation failed; retrying",
			zap.String("operation", operation),
			zap.Duration("retry_in", messageQueueRetryDelay),
			zap.Error(err),
		)
		timer := time.NewTimer(messageQueueRetryDelay)
		select {
		case <-timer.C:
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		}
	}
}
