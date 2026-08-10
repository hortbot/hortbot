// Package conduit implements the main command for the conduit service.
package conduit

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/hortbot/hortbot/internal/cli"
	"github.com/hortbot/hortbot/internal/cli/flags/httpflags"
	"github.com/hortbot/hortbot/internal/cli/flags/promflags"
	"github.com/hortbot/hortbot/internal/cli/flags/sqlflags"
	"github.com/hortbot/hortbot/internal/cli/flags/twitchflags"
	"github.com/hortbot/hortbot/internal/conduit"
	"github.com/hortbot/hortbot/internal/db/chatqueue"
	"github.com/hortbot/hortbot/internal/db/dbsql"
	"github.com/hortbot/hortbot/internal/pkg/apiclient/twitch/eventsub"
	"github.com/hortbot/hortbot/internal/pkg/contextx"
	"github.com/hortbot/hortbot/internal/pkg/errgroupx"
	"github.com/hortbot/hortbot/internal/pkg/eventsubsync"
	"github.com/zikaeroh/ctxlog"
	"go.uber.org/zap"
)

type cmd struct {
	cli.Common
	SQL        sqlflags.SQL
	Twitch     twitchflags.Twitch
	Prometheus promflags.Prometheus
	HTTP       httpflags.HTTP

	SyncInterval time.Duration `long:"conduit-sync-interval" env:"HB_CONDUIT_SYNC_INTERVAL" description:"How often to synchronize subscriptions"`
	Shards       int           `long:"conduit-shards" env:"HB_CONDUIT_SHARDS" description:"Number of shards"`
}

// Command returns a fresh conduit command.
func Command() cli.Command {
	return &cmd{
		Common:       cli.Default,
		SQL:          sqlflags.Default,
		Twitch:       twitchflags.Default,
		Prometheus:   promflags.Default,
		HTTP:         httpflags.Default,
		SyncInterval: 5 * time.Minute,
		Shards:       1,
	}
}

func (*cmd) Name() string {
	return "conduit"
}

func (c *cmd) Main(ctx context.Context, _ []string) {
	c.Prometheus.Run(ctx)

	db := c.SQL.Open(ctx)
	defer db.Close() //nolint:errcheck

	enqueueCtx, cancelEnqueue := contextx.WithGracePeriod(ctx, 30*time.Second)
	defer cancelEnqueue()

	twitchAPI := c.Twitch.Client(c.HTTP.Client())

	g := errgroupx.FromContext(ctx)

	queue := chatqueue.New(db, 1)
	syncRequests := eventsubsync.Requests{}
	queries := dbsql.New(db)
	s := conduit.New(db, twitchAPI, c.SyncInterval, c.Shards, newNotificationHandler(enqueueCtx, queue))

	g.Go(s.Run)
	g.Go(func(ctx context.Context) error {
		return runQueueMaintenance(ctx, queue)
	})

	g.Go(func(ctx context.Context) error {
		syncTicker := time.NewTicker(c.SyncInterval)
		defer syncTicker.Stop()
		requestTicker := time.NewTicker(time.Second)
		defer requestTicker.Stop()

		var handledVersion int64 = -1

		for {
			version, err := syncRequests.Version(ctx, queries)
			if err != nil {
				ctxlog.Error(ctx, "error reading EventSub sync version", zap.Error(err))
				select {
				case <-requestTicker.C:
					continue
				case <-ctx.Done():
					return ctx.Err()
				}
			}
			if version != handledVersion {
				if handledVersion >= 0 {
					timer := time.NewTimer(time.Second)
					select {
					case <-timer.C:
					case <-ctx.Done():
						timer.Stop()
						return ctx.Err()
					}
				}
				if err := s.SynchronizeSubscriptions(ctx); err != nil {
					ctxlog.Error(ctx, "error synchronizing requested EventSub subscriptions", zap.Error(err))
				} else {
					handledVersion = version
				}
			}

			select {
			case <-syncTicker.C:
				if err := s.SynchronizeSubscriptions(ctx); err != nil {
					ctxlog.Error(ctx, "error synchronizing EventSub subscriptions", zap.Error(err))
				}
			case <-requestTicker.C:
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	})

	if err := g.WaitIgnoreStop(); err != nil {
		ctxlog.Info(ctx, "exiting", zap.Error(err))
	}
}

func runQueueMaintenance(ctx context.Context, queue *chatqueue.Queue) error {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for {
		now := time.Now()
		for {
			deleted, err := queue.Cleanup(
				ctx,
				now.Add(-chatqueue.PendingRetention),
				now.Add(-chatqueue.DedupeDuration),
				now.Add(-chatqueue.FailedRetention),
				chatqueue.CleanupBatchSize,
			)
			if err != nil {
				ctxlog.Error(ctx, "error cleaning up chat queue", zap.Error(err))
				break
			}
			if deleted.Stale < chatqueue.CleanupBatchSize &&
				deleted.Completed < chatqueue.CleanupBatchSize &&
				deleted.Failed < chatqueue.CleanupBatchSize {
				break
			}
		}

		select {
		case <-ticker.C:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func newNotificationHandler(enqueueCtx context.Context, queue *chatqueue.Queue) conduit.NotificationHandler {
	return func(ctx context.Context, raw json.RawMessage, message *eventsub.WebsocketMessage) error {
		queued, err := queuedMessage(raw, message)
		if err != nil {
			return err
		}
		for {
			inserted, err := queue.Enqueue(enqueueCtx, queued)
			if err == nil {
				if !inserted {
					ctxlog.Debug(ctx, "duplicate incoming message ignored", zap.String("message_id", queued.ID))
				}
				return nil
			}
			if enqueueCtx.Err() != nil {
				return fmt.Errorf("enqueue incoming message: %w", enqueueCtx.Err())
			}
			ctxlog.Warn(ctx, "error enqueueing incoming message; retrying",
				zap.String("message_id", queued.ID),
				zap.Duration("retry_in", time.Second),
				zap.Error(err),
			)
			timer := time.NewTimer(time.Second)
			select {
			case <-timer.C:
			case <-enqueueCtx.Done():
				timer.Stop()
				return fmt.Errorf("enqueue incoming message: %w", enqueueCtx.Err())
			}
		}
	}
}

func queuedMessage(raw json.RawMessage, m *eventsub.WebsocketMessage) (chatqueue.Message, error) {
	if m == nil || m.Metadata == nil {
		return chatqueue.Message{}, errors.New("incoming message has nil metadata")
	}

	notification, ok := m.Payload.(*eventsub.NotificationPayload)
	if !ok {
		return chatqueue.Message{}, errors.New("incoming message has invalid notification payload")
	}
	event, ok := notification.Event.(*eventsub.ChatMessageEvent)
	if !ok {
		return chatqueue.Message{}, errors.New("incoming message has invalid chat event")
	}
	if m.Metadata.MessageID == "" {
		return chatqueue.Message{}, errors.New("incoming eventsub message has empty message ID")
	}
	if event.MessageID == "" {
		return chatqueue.Message{}, errors.New("incoming chat event has empty message ID")
	}
	if event.BroadcasterUserLogin == "" {
		return chatqueue.Message{}, fmt.Errorf("incoming chat event %q has empty broadcaster login", event.MessageID)
	}
	if m.Metadata.MessageTimestamp.IsZero() {
		return chatqueue.Message{}, fmt.Errorf("incoming chat event %q has zero timestamp", event.MessageID)
	}
	if !json.Valid(raw) {
		return chatqueue.Message{}, fmt.Errorf("incoming chat event %q has invalid raw JSON", event.MessageID)
	}

	return chatqueue.Message{
		ID:               m.Metadata.MessageID,
		BroadcasterLogin: event.BroadcasterUserLogin,
		MessageTimestamp: m.Metadata.MessageTimestamp,
		EnqueuedAt:       time.Now(),
		Payload:          raw,
	}, nil
}
