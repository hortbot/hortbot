// Package bot implements the main command for the bot service.
package bot

import (
	"context"
	"time"

	"github.com/hortbot/hortbot/internal/bnsq"
	"github.com/hortbot/hortbot/internal/cli"
	"github.com/hortbot/hortbot/internal/cli/flags/botflags"
	"github.com/hortbot/hortbot/internal/cli/flags/httpflags"
	"github.com/hortbot/hortbot/internal/cli/flags/jaegerflags"
	"github.com/hortbot/hortbot/internal/cli/flags/nsqflags"
	"github.com/hortbot/hortbot/internal/cli/flags/promflags"
	"github.com/hortbot/hortbot/internal/cli/flags/redisflags"
	"github.com/hortbot/hortbot/internal/cli/flags/sqlflags"
	"github.com/hortbot/hortbot/internal/cli/flags/twitchflags"
	"github.com/hortbot/hortbot/internal/pkg/ctxlog"
	"github.com/hortbot/hortbot/internal/pkg/errgroupx"
	"go.opencensus.io/trace"
	"go.uber.org/zap"
	"golang.org/x/sync/semaphore"
)

type cmd struct {
	cli.Common
	SQL        sqlflags.SQL
	Twitch     twitchflags.Twitch
	Redis      redisflags.Redis
	Bot        botflags.Bot
	NSQ        nsqflags.NSQ
	Jaeger     jaegerflags.Jaeger
	Prometheus promflags.Prometheus
	HTTP       httpflags.HTTP
}

// Command returns a fresh bot command.
func Command() cli.Command {
	return &cmd{
		Common:     cli.Default,
		SQL:        sqlflags.Default,
		Twitch:     twitchflags.Default,
		Redis:      redisflags.Default,
		Bot:        botflags.Default,
		NSQ:        nsqflags.Default,
		Jaeger:     jaegerflags.Default,
		Prometheus: promflags.Default,
		HTTP:       httpflags.Default,
	}
}

func (*cmd) Name() string {
	return "bot"
}

func (c *cmd) Main(ctx context.Context, _ []string) {
	defer c.Jaeger.Init(ctx, c.Name(), c.Debug)()
	c.Prometheus.Run(ctx)

	httpClient := c.HTTP.Client()
	driverName := c.SQL.DriverName()
	driverName = c.Jaeger.DriverName(ctx, driverName, c.Debug)
	db := c.SQL.Open(ctx, driverName)
	rdb := c.Redis.Client()
	twitchAPI := c.Twitch.Client(httpClient)
	sender := c.NSQ.NewSendMessagePublisher()
	notifier := c.NSQ.NewNotifyPublisher()

	b := c.Bot.New(ctx, db, rdb, sender, notifier, twitchAPI, httpClient)
	defer b.Stop()

	sem := semaphore.NewWeighted(int64(c.Bot.Workers))

	g := errgroupx.FromContext(ctx)

	incomingSub := c.NSQ.NewIncomingSubscriber(15*time.Second, func(i *bnsq.Incoming, metadata *bnsq.Metadata) error {
		ctx, span := trace.StartSpanWithRemoteParent(ctx, "OnIncoming", metadata.ParentSpan())
		defer span.End()

		if err := sem.Acquire(ctx, 1); err != nil {
			return err
		}

		g.Go(func(ctx context.Context) error {
			// This context is not the context above.
			ctx = metadata.With(ctx)
			ctx, span := trace.StartSpanWithRemoteParent(ctx, "Worker", span.SpanContext())
			defer span.End()

			defer sem.Release(1)
			b.Handle(ctx, i.Origin, i.Message)
			return ctx.Err()
		})

		return nil
	})

	g.Go(sender.Run)
	g.Go(notifier.Run)
	g.Go(incomingSub.Run)

	if err := g.WaitIgnoreStop(); err != nil {
		ctxlog.Info(ctx, "exiting", zap.Error(err))
	}
}
