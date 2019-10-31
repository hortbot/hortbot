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
	sqlflags.SQL
	twitchflags.Twitch
	redisflags.Redis
	botflags.Bot
	nsqflags.NSQ
	jaegerflags.Jaeger
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
	logger := ctxlog.FromContext(ctx)

	defer config.InitJaeger(ctx, "bot", config.Debug)()

	connector := config.DBConnector(ctx)
	connector = config.TraceDB(config.Debug, connector)
	db := config.OpenDB(ctx, connector)
	rdb := config.RedisClient()
	twitchAPI := config.TwitchClient()
	sender := config.NewSendMessagePublisher()
	notifier := config.NewNotifyPublisher()

	b := config.NewBot(ctx, db, rdb, sender, notifier, twitchAPI)
	defer b.Stop()

	sem := semaphore.NewWeighted(int64(config.Workers))

	g := errgroupx.FromContext(ctx)

	incomingSub := config.NewIncomingSubscriber(15*time.Second, func(i *bnsq.Incoming, parent trace.SpanContext) error {
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
		logger.Info("exiting", zap.Error(err))
	}
}
