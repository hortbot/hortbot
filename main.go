package main

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"os"
	"time"

	"github.com/go-redis/redis/v7"
	"github.com/go-redis/redis_rate/v8"
	"github.com/hortbot/hortbot/internal/birc"
	"github.com/hortbot/hortbot/internal/bot"
	"github.com/hortbot/hortbot/internal/db/migrations"
	"github.com/hortbot/hortbot/internal/db/modelsx"
	"github.com/hortbot/hortbot/internal/pkg/apis/extralife"
	"github.com/hortbot/hortbot/internal/pkg/apis/lastfm"
	"github.com/hortbot/hortbot/internal/pkg/apis/steam"
	"github.com/hortbot/hortbot/internal/pkg/apis/tinyurl"
	"github.com/hortbot/hortbot/internal/pkg/apis/twitch"
	"github.com/hortbot/hortbot/internal/pkg/apis/xkcd"
	"github.com/hortbot/hortbot/internal/pkg/apis/youtube"
	"github.com/hortbot/hortbot/internal/pkg/ctxlog"
	"github.com/hortbot/hortbot/internal/pkg/dedupe/memory"
	"github.com/hortbot/hortbot/internal/pkg/errgroupx"
	"github.com/hortbot/hortbot/internal/pkg/rdb"
	"github.com/hortbot/hortbot/internal/web"
	"github.com/jessevdk/go-flags"
	"github.com/posener/ctxutil"
	"go.uber.org/zap"

	_ "github.com/joho/godotenv/autoload" // Pull .env into env vars.
	_ "github.com/lib/pq"                 // For postgres
)

var args = struct {
	Debug bool `long:"debug" env:"HB_DEBUG" description:"Enables debug mode and the debug log level"`

	Nick string `long:"nick" env:"HB_NICK" description:"IRC nick" required:"true"`
	Pass string `long:"pass" env:"HB_PASS" description:"IRC pass" required:"true"`

	DB        string `long:"db" env:"HB_DB" description:"PostgresSQL connection string" required:"true"`
	MigrateUp bool   `long:"migrate-up" env:"HB_MIGRATE_UP" description:"Migrates the postgres database up"`
	Redis     string `long:"redis" env:"HB_REDIS" description:"Redis address" required:"true"`

	Admins []string `long:"admin" env:"HB_ADMINS" env-delim:"," description:"Bot admins"`

	WhitelistEnabled bool     `long:"whitelist-enabled" env:"HB_WHITELIST_ENABLED" description:"Enable the user whitelist"`
	Whitelist        []string `long:"whitelist" env:"HB_WHITELIST" env-delim:"," description:"User whitelist"`

	DefaultCooldown int    `long:"default-cooldown" env:"HB_DEFAULT_COOLDOWN" description:"default command cooldown"`
	LastFMKey       string `long:"lastfm-key" env:"HB_LASTFM_KEY" description:"LastFM API key"`

	TwitchClientID     string `long:"twitch-client-id" env:"HB_TWITCH_CLIENT_ID" description:"Twitch OAuth client ID" required:"true"`
	TwitchClientSecret string `long:"twitch-client-secret" env:"HB_TWITCH_CLIENT_SECRET" description:"Twitch OAuth client secret" required:"true"`
	TwitchRedirectURL  string `long:"twitch-redirect-url" env:"HB_TWITCH_REDIRECT_URL" description:"Twitch OAuth redirect URL" required:"true"`

	SteamKey string `long:"steam-key" env:"HB_STEAM_KEY" description:"Steam API key"`

	WebAddr string `long:"web-addr" env:"HB_WEB_ADDR" description:"Server address for the web server"`

	RateLimitRate   int           `long:"rate-limit-rate" env:"HB_RATE_LIMIT_RATE" description:"Rate limit rate for sending messages"`
	RateLimitBurst  int           `long:"rate-limit-burst" env:"HB_RATE_LIMIT_BURT" description:"Rate limit burst rate for sending messages"`
	RateLimitPeriod time.Duration `long:"rate-limit-period" env:"HB_RATE_LIMIT_PERIOD" description:"Rate limit period for sending messages"`
}{
	DefaultCooldown: 5,
	WebAddr:         ":5000",
	RateLimitRate:   15,
	RateLimitBurst:  10,
	RateLimitPeriod: 30 * time.Second,
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

	rateLimiter := redis_rate.NewLimiter(rClient, &redis_rate.Limit{
		Rate:   args.RateLimitRate,
		Burst:  args.RateLimitBurst,
		Period: args.RateLimitPeriod,
	})

	botRDB, err := rdb.New(rClient, rdb.KeyPrefix("bot"))
	if err != nil {
		logger.Fatal("error creating RDB", zap.Error(err))
	}

	webRDB, err := rdb.New(rClient, rdb.KeyPrefix("web"))
	if err != nil {
		logger.Fatal("error creating RDB", zap.Error(err))
	}

	channels, err := modelsx.ListActiveChannels(ctx, db, args.Nick)
	if err != nil {
		logger.Fatal("error listing initial channels", zap.Error(err))
	}

	pc := birc.PoolConfig{
		Config: birc.Config{
			UserConfig: birc.UserConfig{
				Nick: args.Nick,
				Pass: args.Pass,
			},
			InitialChannels: channels,
			Caps:            []string{birc.TwitchCapCommands, birc.TwitchCapTags},
		},
	}

	conn := birc.NewPool(pc)

	var sender senderFunc = func(ctx context.Context, origin, target, message string) error {
		for i := 0; i < 2; i++ {
			result, err := rateLimiter.Allow("ratelimit:" + origin)
			if err != nil {
				return err
			}

			if result.Allowed {
				return conn.SendMessage(ctx, target, message)
			}

			ctxlog.FromContext(ctx).Warn("rate limited, sleeping", zap.Duration("sleep", result.RetryAfter))
			time.Sleep(result.RetryAfter)
		}

		return errors.New("rate limited")
	}

	syncJoined := make(chan struct{}, 1)

	var notifier notiferFunc = func(ctx context.Context, botName string) error {
		logger.Debug("notified update to channels for bot", zap.String("botName", botName))
		select {
		case syncJoined <- struct{}{}:
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		return nil
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

	ddp := memory.New(time.Minute, 5*time.Minute)
	defer ddp.Stop()

	bc := &bot.Config{
		DB:               db,
		RDB:              botRDB,
		Dedupe:           ddp,
		Sender:           sender,
		Notifier:         notifier,
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

	g := errgroupx.FromContext(ctx)

	g.Go(func(ctx context.Context) error {
		a := web.App{
			Addr:   args.WebAddr,
			RDB:    webRDB,
			DB:     db,
			Twitch: twitchAPI,
		}

		return a.Run(ctx)
	})

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

				// TODO: Do this in g.Go once there is a channel lock.
				b.Handle(ctx, args.Nick, m)
			}
		}
	})

	g.Go(func(ctx context.Context) error {
		for {
			select {
			case <-ctx.Done():
				return ctx.Err()

			case <-syncJoined:
				time.Sleep(time.Second) // The notification comes in before the transaction is complete.

			case <-time.After(time.Minute):
			}

			channels, err := modelsx.ListActiveChannels(ctx, db, args.Nick)
			if err != nil {
				return err
			}

			if err := conn.SyncJoined(ctx, channels...); err != nil {
				return err
			}
		}
	})

	g.Go(conn.Run)

	if err := g.WaitIgnoreStop(); err != nil {
		logger.Info("exiting", zap.Error(err))
	}
}

type senderFunc func(ctx context.Context, origin, target, message string) error

func (s senderFunc) SendMessage(ctx context.Context, origin, target, message string) error {
	return s(ctx, origin, target, message)
}

type notiferFunc func(ctx context.Context, botName string) error

func (n notiferFunc) NotifyChannelUpdates(ctx context.Context, botName string) error {
	return n(ctx, botName)
}
