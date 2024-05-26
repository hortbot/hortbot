// Package conduit implements the main command for the conduit service.
package conduit

import (
	"context"
	"time"

	"github.com/hortbot/hortbot/internal/bnsq"
	"github.com/hortbot/hortbot/internal/cli"
	"github.com/hortbot/hortbot/internal/cli/flags/httpflags"
	"github.com/hortbot/hortbot/internal/cli/flags/jaegerflags"
	"github.com/hortbot/hortbot/internal/cli/flags/nsqflags"
	"github.com/hortbot/hortbot/internal/cli/flags/promflags"
	"github.com/hortbot/hortbot/internal/cli/flags/sqlflags"
	"github.com/hortbot/hortbot/internal/cli/flags/twitchflags"
	"github.com/hortbot/hortbot/internal/conduit"
	"github.com/hortbot/hortbot/internal/pkg/errgroupx"
	"github.com/zikaeroh/ctxlog"
	"go.opencensus.io/trace"
	"go.uber.org/zap"
)

type cmd struct {
	cli.Common
	SQL        sqlflags.SQL
	Twitch     twitchflags.Twitch
	NSQ        nsqflags.NSQ
	Jaeger     jaegerflags.Jaeger
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
		NSQ:          nsqflags.Default,
		Jaeger:       jaegerflags.Default,
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
	defer c.Jaeger.Trace(ctx, c.Name(), c.Debug)()
	c.Prometheus.Run(ctx)

	driverName := c.SQL.DriverName()
	driverName = c.Jaeger.DriverName(ctx, driverName, c.Debug)
	db := c.SQL.Open(ctx, driverName)

	twitchAPI := c.Twitch.Client(c.HTTP.Client())

	incomingPub := c.NSQ.NewIncomingWebsocketMessagePublisher()

	g := errgroupx.FromContext(ctx)

	s := conduit.New(db, twitchAPI, c.SyncInterval, c.Shards)

	g.Go(s.Run)

	g.Go(func(ctx context.Context) error {
		inc := s.Incoming()

		for {
			select {
			case <-ctx.Done():
				return ctx.Err()

			case m, ok := <-inc:
				if !ok {
					ctxlog.Debug(ctx, "incoming channel closed")
					return nil
				}

				if err := incomingPub.Publish(ctx, m); err != nil {
					ctxlog.Error(ctx, "error publishing incoming message", zap.Error(err))
				}
			}
		}
	})

	syncJoined := make(chan struct{}, 1)

	notifySub := c.NSQ.NewEventsubNotifySubscriber(time.Minute, func(n *bnsq.EventsubNotify, metadata *bnsq.Metadata) error {
		ctx := metadata.With(ctx)
		ctx, span := trace.StartSpanWithRemoteParent(ctx, "OnNotifyChannelUpdates", metadata.ParentSpan())
		defer span.End()

		select {
		case syncJoined <- struct{}{}:
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		return nil
	})

	g.Go(notifySub.Run)
	g.Go(incomingPub.Run)

	g.Go(func(ctx context.Context) error {
		t := time.NewTicker(c.SyncInterval)
		defer t.Stop()

		for {
			// Start with a synchronize, then wait for the interval.
			if err := s.SynchronizeSubscriptions(ctx); err != nil {
				return err
			}

			select {
			case <-t.C:
			case <-syncJoined:
				time.Sleep(time.Second) // The notification comes in before the transaction is complete.
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	})

	if err := g.WaitIgnoreStop(); err != nil {
		ctxlog.Info(ctx, "exiting", zap.Error(err))
	}
}
