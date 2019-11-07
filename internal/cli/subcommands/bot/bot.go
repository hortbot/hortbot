package bot

import (
	"context"
	"time"

	"github.com/hortbot/hortbot/internal/bnsq"
	"github.com/hortbot/hortbot/internal/cli"
	"github.com/hortbot/hortbot/internal/cli/flags/botflags"
	"github.com/hortbot/hortbot/internal/cli/flags/jaegerflags"
	"github.com/hortbot/hortbot/internal/cli/flags/nsqflags"
	"github.com/hortbot/hortbot/internal/cli/flags/redisflags"
	"github.com/hortbot/hortbot/internal/cli/flags/sqlflags"
	"github.com/hortbot/hortbot/internal/cli/flags/twitchflags"
	"github.com/hortbot/hortbot/internal/pkg/ctxlog"
	"github.com/hortbot/hortbot/internal/pkg/errgroupx"
	"go.opencensus.io/trace"
	"go.uber.org/zap"
	"golang.org/x/sync/semaphore"
)

type config struct {
	cli.Common
	SQL    sqlflags.SQL
	Twitch twitchflags.Twitch
	Redis  redisflags.Redis
	Bot    botflags.Bot
	NSQ    nsqflags.NSQ
	Jaeger jaegerflags.Jaeger
}

func Run(args []string) {
	cli.Run("bot", args, &config{
		Common: cli.DefaultCommon,
		SQL:    sqlflags.DefaultSQL,
		Twitch: twitchflags.DefaultTwitch,
		Redis:  redisflags.DefaultRedis,
		Bot:    botflags.DefaultBot,
		NSQ:    nsqflags.DefaultNSQ,
		Jaeger: jaegerflags.DefaultJaeger,
	})
}

func (config *config) Main(ctx context.Context, _ []string) {
	defer config.Jaeger.Init(ctx, "bot", config.Debug)()

	connector := config.SQL.Connector(ctx)
	connector = config.Jaeger.TraceDB(config.Debug, connector)
	db := config.SQL.Open(ctx, connector)
	rdb := config.Redis.Client()
	twitchAPI := config.Twitch.Client()
	sender := config.NSQ.NewSendMessagePublisher()
	notifier := config.NSQ.NewNotifyPublisher()

	b := config.Bot.New(ctx, db, rdb, sender, notifier, twitchAPI)
	defer b.Stop()

	sem := semaphore.NewWeighted(int64(config.Bot.Workers))

	g := errgroupx.FromContext(ctx)

	incomingSub := config.NSQ.NewIncomingSubscriber(15*time.Second, func(i *bnsq.Incoming, parent trace.SpanContext) error {
		ctx, span := trace.StartSpanWithRemoteParent(ctx, "OnIncoming", parent)
		defer span.End()

		if err := sem.Acquire(ctx, 1); err != nil {
			return err
		}

		g.Go(func(ctx context.Context) error {
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
