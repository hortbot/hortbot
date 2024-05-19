// Package irc implements the main command for the IRC service.
package irc

import (
	"context"
	"time"

	"github.com/hortbot/hortbot/internal/bnsq"
	"github.com/hortbot/hortbot/internal/cli"
	"github.com/hortbot/hortbot/internal/cli/flags/httpflags"
	"github.com/hortbot/hortbot/internal/cli/flags/ircflags"
	"github.com/hortbot/hortbot/internal/cli/flags/jaegerflags"
	"github.com/hortbot/hortbot/internal/cli/flags/nsqflags"
	"github.com/hortbot/hortbot/internal/cli/flags/promflags"
	"github.com/hortbot/hortbot/internal/cli/flags/sqlflags"
	"github.com/hortbot/hortbot/internal/cli/flags/twitchflags"
	"github.com/hortbot/hortbot/internal/db/modelsx"
	"github.com/hortbot/hortbot/internal/pkg/errgroupx"
	"github.com/zikaeroh/ctxlog"
	"go.opencensus.io/trace"
	"go.uber.org/zap"
)

type cmd struct {
	cli.Common
	SQL        sqlflags.SQL
	Twitch     twitchflags.Twitch
	IRC        ircflags.IRC
	NSQ        nsqflags.NSQ
	Jaeger     jaegerflags.Jaeger
	Prometheus promflags.Prometheus
	HTTP       httpflags.HTTP
}

// Command returns a fresh irc command.
func Command() cli.Command {
	return &cmd{
		Common:     cli.Default,
		SQL:        sqlflags.Default,
		Twitch:     twitchflags.Default,
		IRC:        ircflags.Default,
		NSQ:        nsqflags.Default,
		Jaeger:     jaegerflags.Default,
		Prometheus: promflags.Default,
		HTTP:       httpflags.Default,
	}
}

func (*cmd) Name() string {
	return "irc"
}

//nolint:gocyclo
func (c *cmd) Main(ctx context.Context, _ []string) {
	defer c.Jaeger.Trace(ctx, c.Name(), c.Debug)()
	c.Prometheus.Run(ctx)

	driverName := c.SQL.DriverName()
	driverName = c.Jaeger.DriverName(ctx, driverName, c.Debug)
	db := c.SQL.Open(ctx, driverName)

	twitchAPI := c.Twitch.Client(c.HTTP.Client())
	conn := c.IRC.Pool(ctx, db, twitchAPI)

	incomingPub := c.NSQ.NewIncomingPublisher()

	syncJoined := make(chan struct{}, 1)

	notifySub := c.NSQ.NewNotifySubscriber(c.IRC.Nick, time.Minute, func(n *bnsq.ChannelUpdatesNotification, metadata *bnsq.Metadata) error {
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

	g := errgroupx.FromContext(ctx)

	g.Go(conn.Run)
	g.Go(incomingPub.Run)

	g.Go(func(ctx context.Context) error {
		inc := conn.Incoming()

		for {
			select {
			case <-ctx.Done():
				return ctx.Err()

			case m, ok := <-inc:
				if !ok {
					return nil
				}

				if err := incomingPub.Publish(ctx, c.IRC.Nick, m); err != nil {
					ctxlog.Error(ctx, "error publishing incoming message", zap.Error(err))
				}
			}
		}
	})

	g.Go(notifySub.Run)

	g.Go(func(ctx context.Context) error {
		for {
			select {
			case <-ctx.Done():
				return ctx.Err()

			case <-syncJoined:
				time.Sleep(time.Second) // The notification comes in before the transaction is complete.

			case <-time.After(time.Minute):
			}

			channels, err := modelsx.ListActiveChannels(ctx, db, c.IRC.Nick)
			if err != nil {
				return err
			}

			if err := conn.SyncJoined(ctx, channels...); err != nil {
				return err
			}
		}
	})

	if err := g.WaitIgnoreStop(); err != nil {
		ctxlog.Info(ctx, "exiting", zap.Error(err))
	}
}
