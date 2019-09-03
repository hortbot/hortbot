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
	"github.com/hortbot/hortbot/internal/bnsq"
	"github.com/hortbot/hortbot/internal/db/migrations"
	"github.com/hortbot/hortbot/internal/db/modelsx"
	"github.com/hortbot/hortbot/internal/db/redis"
	"github.com/hortbot/hortbot/internal/pkg/ctxlog"
	"github.com/hortbot/hortbot/internal/pkg/errgroupx"
	"github.com/hortbot/hortbot/internal/pkg/tracing"
	"github.com/jessevdk/go-flags"
	"github.com/lib/pq"
	"github.com/luna-duclos/instrumentedsql"
	sqltracing "github.com/luna-duclos/instrumentedsql/opentracing"
	"github.com/opentracing/opentracing-go"
	"github.com/posener/ctxutil"
	"go.uber.org/zap"

	_ "github.com/joho/godotenv/autoload" // Pull .env into env vars.
)

var args = struct {
	Nick string `long:"nick" env:"HB_NICK" description:"IRC nick" required:"true"`
	Pass string `long:"pass" env:"HB_PASS" description:"IRC pass" required:"true"`

	DB         string `long:"db" env:"HB_DB" description:"PostgresSQL connection string" required:"true"`
	NSQAddr    string `long:"nsq-addr" env:"HB_NSQ_ADDR" description:"NSQD address" required:"true"`
	NSQChannel string `long:"nsq-channel" env:"HB_NSQ_CHANNEL" description:"NSQ subscription channel"`

	Debug     bool   `long:"debug" env:"HB_DEBUG" description:"Enables debug mode and the debug log level"`
	MigrateUp bool   `long:"migrate-up" env:"HB_MIGRATE_UP" description:"Migrates the postgres database up"`
	Redis     string `long:"redis" env:"HB_REDIS" description:"Redis address" required:"true"`

	RateLimitRate   int           `long:"rate-limit-rate" env:"HB_RATE_LIMIT_RATE" description:"Message allowed per rate limit period"`
	RateLimitPeriod time.Duration `long:"rate-limit-period" env:"HB_RATE_LIMIT_PERIOD" description:"Rate limit period"`
}{
	NSQChannel:      "queue",
	RateLimitRate:   15,
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

	stopTracing, err := tracing.Init("irc", args.Debug, logger)
	if err != nil {
		logger.Fatal("error initializing tracing", zap.Error(err))
	}
	defer stopTracing.Close()

	sql.Register("postgres-opentracing",
		instrumentedsql.WrapDriver(&pq.Driver{},
			instrumentedsql.WithTracer(sqltracing.NewTracer(true)),
			instrumentedsql.WithOmitArgs(),
		),
	)

	db, err := sql.Open("postgres-opentracing", args.DB)
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
		Addr: args.Redis,
	})
	defer rClient.Close()

	rdb := redis.New(rClient)

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

	incomingPub := bnsq.NewIncomingPublisher(args.NSQAddr)

	sendSub := &bnsq.SendMessageSubscriber{
		Addr:    args.NSQAddr,
		Origin:  args.Nick,
		Channel: args.NSQChannel,
		Opts: []bnsq.SubscriberOption{
			bnsq.SubscriberMaxAge(5 * time.Second),
		},
		OnSendMessage: func(m *bnsq.SendMessage, ref opentracing.SpanReference) error {
			span, ctx := opentracing.StartSpanFromContext(ctx, "OnSendMessage", ref)
			defer span.Finish()

			allowed, err := rdb.SendMessageAllowed(ctx, m.Origin, args.RateLimitRate, args.RateLimitPeriod)
			if err != nil {
				logger.Error("error checking rate limit", zap.Error(err))
				return err
			}

			if !allowed {
				logger.Error("rate limited, requeueing")
				return errors.New("rate limited")
			}

			if err := conn.SendMessage(ctx, m.Target, m.Message); err != nil {
				logger.Error("error sending message", zap.Error(err))
				return err
			}

			return nil
		},
	}

	syncJoined := make(chan struct{}, 1)

	notifySub := &bnsq.NotifySubscriber{
		Addr:    args.NSQAddr,
		BotName: args.Nick,
		Channel: args.NSQChannel,
		Opts: []bnsq.SubscriberOption{
			bnsq.SubscriberMaxAge(time.Minute),
		},
		OnNotifyChannelUpdates: func(n *bnsq.ChannelUpdatesNotification, ref opentracing.SpanReference) error {
			span, ctx := opentracing.StartSpanFromContext(ctx, "OnNotifyChannelUpdates", ref)
			defer span.Finish()

			select {
			case syncJoined <- struct{}{}:
			case <-ctx.Done():
				return ctx.Err()
			default:
			}

			return nil
		},
	}

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

				if err := incomingPub.Publish(ctx, args.Nick, m); err != nil {
					logger.Error("error publishing incoming message", zap.Error(err))
				}
			}
		}
	})

	g.Go(sendSub.Run)
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

			channels, err := modelsx.ListActiveChannels(ctx, db, args.Nick)
			if err != nil {
				return err
			}

			if err := conn.SyncJoined(ctx, channels...); err != nil {
				return err
			}
		}
	})

	if err := g.WaitIgnoreStop(); err != nil {
		logger.Info("exiting", zap.Error(err))
	}
}
