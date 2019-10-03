package main

import (
	"context"
	"errors"
	"time"

	"github.com/hortbot/hortbot/internal/cmdargs"
	"github.com/hortbot/hortbot/internal/cmdargs/botargs"
	"github.com/hortbot/hortbot/internal/cmdargs/ircargs"
	"github.com/hortbot/hortbot/internal/cmdargs/redisargs"
	"github.com/hortbot/hortbot/internal/cmdargs/rlargs"
	"github.com/hortbot/hortbot/internal/cmdargs/sqlargs"
	"github.com/hortbot/hortbot/internal/cmdargs/twitchargs"
	"github.com/hortbot/hortbot/internal/cmdargs/webargs"
	"github.com/hortbot/hortbot/internal/db/modelsx"
	"github.com/hortbot/hortbot/internal/pkg/ctxlog"
	"github.com/hortbot/hortbot/internal/pkg/errgroupx"
	"go.uber.org/zap"
)

var args = struct {
	cmdargs.Common
	sqlargs.SQL
	twitchargs.Twitch
	ircargs.IRC
	redisargs.Redis
	webargs.Web
	botargs.Bot
	rlargs.RateLimit
}{
	Common:    cmdargs.DefaultCommon,
	SQL:       sqlargs.DefaultSQL,
	Twitch:    twitchargs.DefaultTwitch,
	IRC:       ircargs.DefaultIRC,
	Redis:     redisargs.DefaultRedis,
	Web:       webargs.DefaultWeb,
	Bot:       botargs.DefaultBot,
	RateLimit: rlargs.DefaultRateLimit,
}

func main() {
	cmdargs.Run(&args, mainCtx)
}

//nolint:gocyclo
func mainCtx(ctx context.Context) {
	logger := ctxlog.FromContext(ctx)

	db := args.OpenDB(ctx, args.DBConnector(ctx))
	rdb := args.RedisClient()
	twitchAPI := args.TwitchClient()
	conn := args.IRCPool(ctx, db, twitchAPI)
	a := args.WebApp(args.Debug, rdb, db, twitchAPI)

	var sender senderFunc = func(ctx context.Context, origin, target, message string) error {
		allowed, err := args.SendMessageAllowed(ctx, rdb, origin, target)
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

	b := args.NewBot(ctx, db, rdb, sender, notifier, twitchAPI)
	defer b.Stop()

	g := errgroupx.FromContext(ctx)

	g.Go(a.Run)

	for i := 0; i < args.Workers; i++ {
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
					b.Handle(ctx, args.Nick, m)
				}
			}
		})
	}

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
