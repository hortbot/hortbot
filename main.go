package main

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"os"
	"time"

	goredis "github.com/go-redis/redis/v7"
	"github.com/hortbot/hortbot/internal/birc"
	"github.com/hortbot/hortbot/internal/bot"
	"github.com/hortbot/hortbot/internal/cmdargs"
	"github.com/hortbot/hortbot/internal/db/migrations"
	"github.com/hortbot/hortbot/internal/db/models"
	"github.com/hortbot/hortbot/internal/db/modelsx"
	"github.com/hortbot/hortbot/internal/db/redis"
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
	"github.com/hortbot/hortbot/internal/web"
	"github.com/jessevdk/go-flags"
	"github.com/posener/ctxutil"
	"github.com/volatiletech/null"
	"go.uber.org/zap"

	_ "github.com/joho/godotenv/autoload" // Pull .env into env vars.
	_ "github.com/lib/pq"                 // For postgres
)

var args = struct {
	cmdargs.Common
	cmdargs.IRC
	cmdargs.SQL
	cmdargs.Redis
	cmdargs.Bot
	cmdargs.LastFM
	cmdargs.Twitch
	cmdargs.Steam
	cmdargs.Web
	cmdargs.RateLimit
}{
	Common:    cmdargs.DefaultCommon,
	IRC:       cmdargs.DefaultIRC,
	SQL:       cmdargs.DefaultSQL,
	Redis:     cmdargs.DefaultRedis,
	Bot:       cmdargs.DefaultBot,
	LastFM:    cmdargs.DefaultLastFM,
	Twitch:    cmdargs.DefaultTwitch,
	Steam:     cmdargs.DefaultSteam,
	Web:       cmdargs.DefaultWeb,
	RateLimit: cmdargs.DefaultRateLimit,
}

//nolint:gocyclo
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

	rClient := goredis.NewClient(&goredis.Options{
		Addr: args.RedisAddr,
	})
	defer rClient.Close()

	rdb := redis.New(rClient)

	channels, err := modelsx.ListActiveChannels(ctx, db, args.Nick)
	if err != nil {
		logger.Fatal("error listing initial channels", zap.Error(err))
	}

	nick := args.Nick
	pass := args.Pass

	token, err := models.TwitchTokens(models.TwitchTokenWhere.BotName.EQ(null.StringFrom(nick))).One(ctx, db)
	if err != nil {
		if err != sql.ErrNoRows {
			logger.Fatal("error querying for bot token", zap.Error(err))
		}
	} else {
		logger.Debug("using oauth token from database")
		pass = "oauth:" + token.AccessToken
	}

	pc := birc.PoolConfig{
		Config: birc.Config{
			UserConfig: birc.UserConfig{
				Nick: nick,
				Pass: pass,
			},
			InitialChannels: channels,
			Caps:            []string{birc.TwitchCapCommands, birc.TwitchCapTags},
			PingInterval:    args.PingInterval,
			PingDeadline:    args.PingDeadline,
		},
	}

	conn := birc.NewPool(pc)

	var sender senderFunc = func(ctx context.Context, origin, target, message string) error {
		allowed, err := rdb.SendMessageAllowed(ctx, origin, target, args.RateLimitSlow, args.RateLimitFast, args.RateLimitPeriod)
		if err != nil {
			return err
		}

		if allowed {
			return conn.SendMessage(ctx, target, message)
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
		Redis:            rdb,
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
			Addr:       args.WebAddr,
			SessionKey: []byte(args.WebSessionKey),
			Brand:      args.WebBrand,
			BrandMap:   args.WebBrandMap,
			Debug:      args.Debug,
			Redis:      rdb,
			DB:         db,
			Twitch:     twitchAPI,
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
