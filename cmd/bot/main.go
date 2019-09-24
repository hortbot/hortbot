package main

import (
	"context"
	"time"

	"github.com/hortbot/hortbot/internal/bnsq"
	"github.com/hortbot/hortbot/internal/cmdargs"
	"github.com/hortbot/hortbot/internal/cmdargs/botargs"
	"github.com/hortbot/hortbot/internal/cmdargs/jaegerargs"
	"github.com/hortbot/hortbot/internal/cmdargs/nsqargs"
	"github.com/hortbot/hortbot/internal/cmdargs/redisargs"
	"github.com/hortbot/hortbot/internal/cmdargs/sqlargs"
	"github.com/hortbot/hortbot/internal/cmdargs/twitchargs"
	"github.com/hortbot/hortbot/internal/pkg/ctxlog"
	"github.com/hortbot/hortbot/internal/pkg/errgroupx"
	"go.opencensus.io/trace"
	"go.uber.org/zap"
)

var args = struct {
	cmdargs.Common
	sqlargs.SQL
	twitchargs.Twitch
	redisargs.Redis
	botargs.Bot
	nsqargs.NSQ
	jaegerargs.Jaeger
}{
	Common: cmdargs.DefaultCommon,
	SQL:    sqlargs.DefaultSQL,
	Twitch: twitchargs.DefaultTwitch,
	Redis:  redisargs.DefaultRedis,
	Bot:    botargs.DefaultBot,
	NSQ:    nsqargs.DefaultNSQ,
	Jaeger: jaegerargs.DefaultJaeger,
}

func main() {
	cmdargs.Run(&args, mainCtx)
}

func mainCtx(ctx context.Context) {
	logger := ctxlog.FromContext(ctx)

	defer args.InitJaeger(ctx, "bot", args.Debug)()

	connector := args.DBConnector(ctx)
	connector = args.TraceDB(args.Debug, connector)
	db := args.OpenDB(ctx, connector)
	rdb := args.RedisClient()
	twitchAPI := args.TwitchClient()
	sender := args.NewSendMessagePublisher()
	notifier := args.NewNotifyPublisher()

	b := args.NewBot(ctx, db, rdb, sender, notifier, twitchAPI)
	defer b.Stop()

	incomingSub := args.NewIncomingSubscriber(5*time.Second, func(i *bnsq.Incoming, parent trace.SpanContext) error {
		ctx, span := trace.StartSpanWithRemoteParent(ctx, "OnIncoming", parent)
		defer span.End()
		b.Handle(ctx, i.Origin, i.Message)
		return nil
	})

	g := errgroupx.FromContext(ctx)
	g.Go(sender.Run)
	g.Go(notifier.Run)
	g.Go(incomingSub.Run)

	if err := g.WaitIgnoreStop(); err != nil {
		logger.Info("exiting", zap.Error(err))
	}
}
