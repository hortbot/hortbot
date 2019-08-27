package main

import (
	"database/sql"
	"log"
	"os"
	"time"

	"github.com/go-redis/redis/v7"
	"github.com/hortbot/hortbot/internal/bnsq"
	"github.com/hortbot/hortbot/internal/bot"
	"github.com/hortbot/hortbot/internal/db/migrations"
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
	"github.com/hortbot/hortbot/internal/pkg/rdb"
	"github.com/hortbot/hortbot/internal/pkg/tracing"
	"github.com/jessevdk/go-flags"
	"github.com/opentracing/opentracing-go"
	"github.com/posener/ctxutil"
	"go.uber.org/zap"

	_ "github.com/joho/godotenv/autoload" // Pull .env into env vars.
	_ "github.com/lib/pq"                 // For postgres
)

var args = struct {
	DB         string `long:"db" env:"HB_DB" description:"PostgresSQL connection string" required:"true"`
	Redis      string `long:"redis" env:"HB_REDIS" description:"Redis address" required:"true"`
	NSQAddr    string `long:"nsq-addr" env:"HB_NSQ_ADDR" description:"NSQD address" required:"true"`
	NSQChannel string `long:"nsq-channel" env:"HB_NSQ_CHANNEL" description:"NSQ subscription channel"`

	Admins []string `long:"admin" env:"HB_ADMINS" env-delim:"," description:"Bot admins"`

	WhitelistEnabled bool     `long:"whitelist-enabled" env:"HB_WHITELIST_ENABLED" description:"Enable the user whitelist"`
	Whitelist        []string `long:"whitelist" env:"HB_WHITELIST" env-delim:"," description:"User whitelist"`

	DefaultCooldown int `long:"default-cooldown" env:"HB_DEFAULT_COOLDOWN" description:"default command cooldown"`

	Debug     bool `long:"debug" env:"HB_DEBUG" description:"Enables debug mode and the debug log level"`
	MigrateUp bool `long:"migrate-up" env:"HB_MIGRATE_UP" description:"Migrates the postgres database up"`

	LastFMKey string `long:"lastfm-key" env:"HB_LASTFM_KEY" description:"LastFM API key"`

	TwitchClientID     string `long:"twitch-client-id" env:"HB_TWITCH_CLIENT_ID" description:"Twitch OAuth client ID" required:"true"`
	TwitchClientSecret string `long:"twitch-client-secret" env:"HB_TWITCH_CLIENT_SECRET" description:"Twitch OAuth client secret" required:"true"`
	TwitchRedirectURL  string `long:"twitch-redirect-url" env:"HB_TWITCH_REDIRECT_URL" description:"Twitch OAuth redirect URL" required:"true"`

	SteamKey string `long:"steam-key" env:"HB_STEAM_KEY" description:"Steam API key"`
}{
	DefaultCooldown: 5,
	NSQChannel:      "queue",
}

func main() {
	ctx := ctxutil.Interrupt()

	if _, err := flags.Parse(&args); err != nil {
		if !flags.WroteHelp(err) {
			log.Fatalln(err)
		}
		os.Exit(1)
	}

	logger := ctxlog.New(args.Debug)
	defer zap.RedirectStdLog(logger)()
	ctx = ctxlog.WithLogger(ctx, logger)

	stopTracing, err := tracing.Init("bot", args.Debug, logger)
	if err != nil {
		logger.Fatal("error initializing tracing", zap.Error(err))
	}
	defer stopTracing.Close()

	db, err := sql.Open("postgres", args.DB)
	if err != nil {
		logger.Fatal("error opening database connection", zap.Error(err))
	}

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

	rClient := redis.NewClient(&redis.Options{
		Addr: args.Redis,
	})
	defer rClient.Close()

	botRDB, err := rdb.New(rClient, rdb.KeyPrefix("bot"))
	if err != nil {
		logger.Fatal("error creating RDB", zap.Error(err))
	}

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

	ddp, err := deduperedis.New(rClient, 5*time.Minute)
	if err != nil {
		logger.Fatal("error making redis dedupe", zap.Error(err))
	}

	sendPub := bnsq.NewSendMessagePublisher(args.NSQAddr)
	notifyPub := bnsq.NewNotifyPublisher(args.NSQAddr)

	bc := &bot.Config{
		DB:               db,
		RDB:              botRDB,
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
		OnIncoming: func(i *bnsq.Incoming, ref opentracing.SpanReference) {
			span, ctx := opentracing.StartSpanFromContext(ctx, "OnIncoming", ref)
			defer span.Finish()
			b.Handle(ctx, i.Origin, i.Message)
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
