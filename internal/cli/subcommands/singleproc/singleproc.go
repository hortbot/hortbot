package singleproc

import (
	"context"
	"errors"
	"time"

	"github.com/hortbot/hortbot/internal/cli"
	"github.com/hortbot/hortbot/internal/cli/flags/botflags"
	"github.com/hortbot/hortbot/internal/cli/flags/ircflags"
	"github.com/hortbot/hortbot/internal/cli/flags/redisflags"
	"github.com/hortbot/hortbot/internal/cli/flags/sqlflags"
	"github.com/hortbot/hortbot/internal/cli/flags/twitchflags"
	"github.com/hortbot/hortbot/internal/cli/flags/webflags"
	"github.com/hortbot/hortbot/internal/db/modelsx"
	"github.com/hortbot/hortbot/internal/pkg/ctxlog"
	"github.com/hortbot/hortbot/internal/pkg/errgroupx"
	"go.uber.org/zap"
)

type cmd struct {
	cli.Common
	sqlflags.SQL
	twitchflags.Twitch
	ircflags.IRC
	redisflags.Redis
	webflags.Web
	botflags.Bot
}

func Run(args []string) {
	cli.Run("single-proc", args, &cmd{
		Common: cli.DefaultCommon,
		SQL:    sqlflags.DefaultSQL,
		Twitch: twitchflags.DefaultTwitch,
		IRC:    ircflags.DefaultIRC,
		Redis:  redisflags.DefaultRedis,
		Web:    webflags.DefaultWeb,
		Bot:    botflags.DefaultBot,
	})
}

//nolint:gocyclo
func (cmd *cmd) Main(ctx context.Context, _ []string) {
	logger := ctxlog.FromContext(ctx)

	db := cmd.OpenDB(ctx, cmd.DBConnector(ctx))
	rdb := cmd.RedisClient()
	twitchAPI := cmd.TwitchClient()
	conn := cmd.IRCPool(ctx, db, twitchAPI)
	a := cmd.WebApp(cmd.Debug, rdb, db, twitchAPI)

	var sender senderFunc = func(ctx context.Context, origin, target, message string) error {
		allowed, err := cmd.SendMessageAllowed(ctx, rdb, origin, target)
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

	b := cmd.NewBot(ctx, db, rdb, sender, notifier, twitchAPI)
	defer b.Stop()

	g := errgroupx.FromContext(ctx)

	g.Go(a.Run)

	for i := 0; i < cmd.Workers; i++ {
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
					b.Handle(ctx, cmd.Nick, m)
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

			channels, err := modelsx.ListActiveChannels(ctx, db, cmd.Nick)
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
