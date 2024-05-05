// Package bot implements the main command for the bot service.
package bot

import (
	"context"
	"runtime"
	"strings"
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
	"github.com/hortbot/hortbot/internal/pkg/errgroupx"
	"github.com/hortbot/hortbot/internal/pkg/wqueue"
	"github.com/jakebailey/irc"
	"github.com/zikaeroh/ctxlog"
	"go.opencensus.io/trace"
	"go.uber.org/zap"
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
	defer c.Jaeger.Trace(ctx, c.Name(), c.Debug)()
	c.Prometheus.Run(ctx)

	httpClient := c.HTTP.Client()
	untrustedClient := c.HTTP.UntrustedClient(ctx)
	driverName := c.SQL.DriverName()
	driverName = c.Jaeger.DriverName(ctx, driverName, c.Debug)
	db := c.SQL.Open(ctx, driverName)
	rdb := c.Redis.Client()
	twitchAPI := c.Twitch.Client(httpClient)
	sender := c.NSQ.NewSendMessagePublisher()
	notifier := c.NSQ.NewNotifyPublisher()

	b := c.Bot.New(ctx, db, rdb, sender, notifier, twitchAPI, httpClient, untrustedClient)
	defer b.Stop()

	g := errgroupx.FromContext(ctx)

	workers := c.Bot.Workers
	if workers < 1 {
		workers = runtime.GOMAXPROCS(0)
	}

	// TODO: pass the queue down to the bot to use internally
	queue := wqueue.NewQueue[string](10 * workers)
	for range workers {
		g.Go(queue.Worker)
	}

	incomingSub := c.NSQ.NewIncomingSubscriber(15*time.Second, func(i *bnsq.Incoming, metadata *bnsq.Metadata) error {
		subCtx, span := trace.StartSpanWithRemoteParent(ctx, "OnIncoming", metadata.ParentSpan())
		defer span.End()

		origin := i.Origin
		m := i.Message
		key := buildKey(m)

		return queue.Put(subCtx, key, func(attach wqueue.Attacher) {
			ctx, cancel := attach(ctx)
			defer cancel()

			ctx = metadata.With(ctx)
			ctx, span := trace.StartSpanWithRemoteParent(ctx, "Worker", span.SpanContext())
			defer span.End()

			b.Handle(ctx, origin, m)
		})
	})

	g.Go(sender.Run)
	g.Go(notifier.Run)
	g.Go(incomingSub.Run)

	if err := g.WaitIgnoreStop(); err != nil {
		ctxlog.Info(ctx, "exiting", zap.Error(err))
	}
}

func buildKey(m *irc.Message) string {
	var keyBuilder strings.Builder

	size := len(m.Command)
	for _, p := range m.Params {
		size += len(p) + 1
	}
	keyBuilder.Grow(size)

	keyBuilder.WriteString(m.Command)
	for _, p := range m.Params {
		keyBuilder.WriteByte(' ')
		keyBuilder.WriteString(p)
	}

	return keyBuilder.String()
}
