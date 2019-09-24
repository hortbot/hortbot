package main

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"contrib.go.opencensus.io/integrations/ocsql"
	goredis "github.com/go-redis/redis/v7"
	"github.com/hortbot/hortbot/internal/birc"
	"github.com/hortbot/hortbot/internal/bnsq"
	"github.com/hortbot/hortbot/internal/cmdargs"
	"github.com/hortbot/hortbot/internal/db/migrations"
	"github.com/hortbot/hortbot/internal/db/modelsx"
	"github.com/hortbot/hortbot/internal/db/redis"
	"github.com/hortbot/hortbot/internal/pkg/apis/twitch"
	"github.com/hortbot/hortbot/internal/pkg/ctxlog"
	"github.com/hortbot/hortbot/internal/pkg/errgroupx"
	"github.com/hortbot/hortbot/internal/pkg/tracing"
	"github.com/hortbot/hortbot/internal/pkg/twitchx"
	"github.com/lib/pq"
	"go.opencensus.io/trace"
	"go.uber.org/zap"

	_ "github.com/joho/godotenv/autoload" // Pull .env into env vars.
)

var args = struct {
	cmdargs.Common
	cmdargs.IRC
	cmdargs.SQL
	cmdargs.Redis
	cmdargs.Twitch
	cmdargs.RateLimit
	cmdargs.NSQ
	cmdargs.Jaeger
}{
	Common:    cmdargs.DefaultCommon,
	IRC:       cmdargs.DefaultIRC,
	SQL:       cmdargs.DefaultSQL,
	Redis:     cmdargs.DefaultRedis,
	Twitch:    cmdargs.DefaultTwitch,
	RateLimit: cmdargs.DefaultRateLimit,
	NSQ:       cmdargs.DefaultNSQ,
	Jaeger:    cmdargs.DefaultJaeger,
}

func main() {
	cmdargs.Run(&args, mainCtx)
}

//nolint:gocyclo
func mainCtx(ctx context.Context) {
	logger := ctxlog.FromContext(ctx)

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

	twitchAPI := twitch.New(args.TwitchClientID, args.TwitchClientSecret, args.TwitchRedirectURL)

	if !args.NoToken {
		token, err := twitchx.FindBotToken(ctx, db, twitchAPI, nick)
		if err != nil {
			logger.Fatal("error querying for bot token", zap.Error(err))
		}
		if token != nil {
			logger.Debug("using token from database")
			pass = "oauth:" + token.AccessToken
		}
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
