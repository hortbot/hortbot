package main

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"os"
	"time"

	"contrib.go.opencensus.io/integrations/ocsql"
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
	"github.com/posener/ctxutil"
	"go.opencensus.io/trace"
	"go.uber.org/zap"

	_ "github.com/joho/godotenv/autoload" // Pull .env into env vars.
)

var args = struct {
	Nick         string        `long:"nick" env:"HB_NICK" description:"IRC nick" required:"true"`
	Pass         string        `long:"pass" env:"HB_PASS" description:"IRC pass" required:"true"`
	PingInterval time.Duration `long:"ping-interval" env:"HB_PING_INTERVAL" description:"How often to ping the IRC server"`
	PingDeadline time.Duration `long:"ping-deadline" env:"HB_PING_DEADLINE" description:"How long to wait for a PONG before disconnecting"`

	DB         string `long:"db" env:"HB_DB" description:"PostgresSQL connection string" required:"true"`
	NSQAddr    string `long:"nsq-addr" env:"HB_NSQ_ADDR" description:"NSQD address" required:"true"`
	NSQChannel string `long:"nsq-channel" env:"HB_NSQ_CHANNEL" description:"NSQ subscription channel"`

	Debug     bool   `long:"debug" env:"HB_DEBUG" description:"Enables debug mode and the debug log level"`
	MigrateUp bool   `long:"migrate-up" env:"HB_MIGRATE_UP" description:"Migrates the postgres database up"`
	Redis     string `long:"redis" env:"HB_REDIS" description:"Redis address" required:"true"`

	RateLimitSlow   int           `long:"rate-limit-slow" env:"HB_RATE_LIMIT_RATE" description:"Message allowed per rate limit period (slow)"`
	RateLimitFast   int           `long:"rate-limit-fast" env:"HB_RATE_LIMIT_RATE" description:"Message allowed per rate limit period (fast)"`
	RateLimitPeriod time.Duration `long:"rate-limit-period" env:"HB_RATE_LIMIT_PERIOD" description:"Rate limit period"`

	JaegerAgent string `long:"jaeger-agent" env:"HB_JAEGER_AGENT" description:"jaeger agent address"`
}{
	PingInterval:    5 * time.Minute,
	PingDeadline:    5 * time.Second,
	NSQChannel:      "queue",
	RateLimitSlow:   15,
	RateLimitFast:   80,
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

	if args.JaegerAgent != "" {
		flush, err := tracing.Init("irc", args.JaegerAgent, args.Debug)
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
			PingInterval:    args.PingInterval,
			PingDeadline:    args.PingDeadline,
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
		OnSendMessage: func(m *bnsq.SendMessage, parent trace.SpanContext) error {
			ctx, span := trace.StartSpanWithRemoteParent(ctx, "OnSendMessage", parent)
			defer span.End()

			allowed, err := rdb.SendMessageAllowed(ctx, m.Origin, m.Target, args.RateLimitSlow, args.RateLimitFast, args.RateLimitPeriod)
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
		OnNotifyChannelUpdates: func(n *bnsq.ChannelUpdatesNotification, parent trace.SpanContext) error {
			ctx, span := trace.StartSpanWithRemoteParent(ctx, "OnNotifyChannelUpdates", parent)
			defer span.End()

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
