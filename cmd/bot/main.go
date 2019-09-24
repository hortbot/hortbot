package main

import (
	"context"
	"database/sql"
	"time"

	"contrib.go.opencensus.io/integrations/ocsql"
	goredis "github.com/go-redis/redis/v7"
	"github.com/hortbot/hortbot/internal/bnsq"
	"github.com/hortbot/hortbot/internal/bot"
	"github.com/hortbot/hortbot/internal/cmdargs"
	"github.com/hortbot/hortbot/internal/db/migrations"
	"github.com/hortbot/hortbot/internal/db/redis"
	"github.com/hortbot/hortbot/internal/pkg/apis/extralife"
	"github.com/hortbot/hortbot/internal/pkg/apis/lastfm"
	"github.com/hortbot/hortbot/internal/pkg/apis/steam"
	"github.com/hortbot/hortbot/internal/pkg/apis/tinyurl"
	"github.com/hortbot/hortbot/internal/pkg/apis/twitch"
	"github.com/hortbot/hortbot/internal/pkg/apis/xkcd"
	"github.com/hortbot/hortbot/internal/pkg/apis/youtube"
	"github.com/hortbot/hortbot/internal/pkg/ctxlog"
	deduperedis "github.com/hortbot/hortbot/internal/pkg/dedupe/redis"
	"github.com/hortbot/hortbot/internal/pkg/errgroupx"
	"github.com/hortbot/hortbot/internal/pkg/tracing"
	"github.com/lib/pq"
	"go.opencensus.io/trace"
	"go.uber.org/zap"

	_ "github.com/joho/godotenv/autoload" // Pull .env into env vars.
)

var args = struct {
	cmdargs.Common
	cmdargs.SQL
	cmdargs.Redis
	cmdargs.Bot
	cmdargs.LastFM
	cmdargs.Twitch
	cmdargs.Steam
	cmdargs.NSQ
	cmdargs.Jaeger
}{
	Common: cmdargs.DefaultCommon,
	SQL:    cmdargs.DefaultSQL,
	Redis:  cmdargs.DefaultRedis,
	Bot:    cmdargs.DefaultBot,
	LastFM: cmdargs.DefaultLastFM,
	Twitch: cmdargs.DefaultTwitch,
	Steam:  cmdargs.DefaultSteam,
	NSQ:    cmdargs.DefaultNSQ,
	Jaeger: cmdargs.DefaultJaeger,
}

func main() {
	cmdargs.Run(&args, mainCtx)
}

func mainCtx(ctx context.Context) {
	logger := ctxlog.FromContext(ctx)

	if args.JaegerAgent != "" {
		flush, err := tracing.Init("bot", args.JaegerAgent, args.Debug)
		if err != nil {
			logger.Fatal("error initializing tracing", zap.Error(err))
		}
		defer flush()
	}

	connector, err := pq.NewConnector(args.DB)
	if err != nil {
		logger.Fatal("error creating postgres connector", zap.Error(err))
	}

	db := sql.OpenDB(ocsql.WrapConnector(connector, ocsql.WithAllTraceOptions(), ocsql.WithQueryParams(args.Debug)))

	for i := 0; i < 5; i++ {
		if err := db.Ping(); err == nil {
			break
		}

		time.Sleep(100 * time.Millisecond)
	}

	if args.MigrateUp {
		if err := migrations.Up(args.DB, nil); err != nil {
			logger.Fatal("error migrating database", zap.Error(err))
		}
	}

	rClient := goredis.NewClient(&goredis.Options{
		Addr: args.RedisAddr,
	})
	defer rClient.Close()

	rdb := redis.New(rClient)

	var lastFM lastfm.API

	if args.LastFMKey != "" {
		lastFM = lastfm.New(args.LastFMKey)
	} else {
		logger.Warn("no LastFM API key provided, functionality will be disabled")
	}

	twitchAPI := twitch.New(args.TwitchClientID, args.TwitchClientSecret, args.TwitchRedirectURL)

	var steamAPI steam.API
	if args.SteamKey != "" {
		steamAPI = steam.New(args.SteamKey)
	} else {
		logger.Warn("no Steam API key provided, functionality will be disabled")
	}

	ddp, err := deduperedis.New(rdb, 5*time.Minute)
	if err != nil {
		logger.Fatal("error making redis dedupe", zap.Error(err))
	}

	sendPub := bnsq.NewSendMessagePublisher(args.NSQAddr)
	notifyPub := bnsq.NewNotifyPublisher(args.NSQAddr)

	bc := &bot.Config{
		DB:               db,
		Redis:            rdb,
		Dedupe:           ddp,
		Sender:           sendPub,
		Notifier:         notifyPub,
		LastFM:           lastFM,
		YouTube:          youtube.New(),
		XKCD:             xkcd.New(),
		ExtraLife:        extralife.New(),
		Twitch:           twitchAPI,
		Steam:            steamAPI,
		TinyURL:          tinyurl.New(),
		Admins:           args.Admins,
		WhitelistEnabled: args.WhitelistEnabled,
		Whitelist:        args.Whitelist,
		Cooldown:         args.DefaultCooldown,
		WebAddr:          args.BotWebAddr,
		WebAddrMap:       args.BotWebAddrMap,
	}

	b := bot.New(bc)

	if err := b.Init(ctx); err != nil {
		logger.Fatal("error initializing bot", zap.Error(err))
	}

	defer b.Stop()

	incomingSub := bnsq.IncomingSubscriber{
		Addr:    args.NSQAddr,
		Channel: args.NSQChannel,
		Opts: []bnsq.SubscriberOption{
			bnsq.SubscriberMaxAge(5 * time.Second),
		},
		OnIncoming: func(i *bnsq.Incoming, parent trace.SpanContext) error {
			ctx, span := trace.StartSpanWithRemoteParent(ctx, "OnIncoming", parent)
			defer span.End()
			b.Handle(ctx, i.Origin, i.Message)
			return nil
		},
	}

	g := errgroupx.FromContext(ctx)
	g.Go(sendPub.Run)
	g.Go(notifyPub.Run)
	g.Go(incomingSub.Run)

	if err := g.WaitIgnoreStop(); err != nil {
		logger.Info("exiting", zap.Error(err))
	}
}
