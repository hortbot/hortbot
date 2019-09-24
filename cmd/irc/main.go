package main

import (
	"context"
	"errors"
	"time"

	"github.com/hortbot/hortbot/internal/bnsq"
	"github.com/hortbot/hortbot/internal/cmdargs"
	"github.com/hortbot/hortbot/internal/cmdargs/ircargs"
	"github.com/hortbot/hortbot/internal/cmdargs/jaegerargs"
	"github.com/hortbot/hortbot/internal/cmdargs/nsqargs"
	"github.com/hortbot/hortbot/internal/cmdargs/redisargs"
	"github.com/hortbot/hortbot/internal/cmdargs/rlargs"
	"github.com/hortbot/hortbot/internal/cmdargs/sqlargs"
	"github.com/hortbot/hortbot/internal/cmdargs/twitchargs"
	"github.com/hortbot/hortbot/internal/db/modelsx"
	"github.com/hortbot/hortbot/internal/pkg/ctxlog"
	"github.com/hortbot/hortbot/internal/pkg/errgroupx"
	"go.opencensus.io/trace"
	"go.uber.org/zap"
)

var args = struct {
	cmdargs.Common
	sqlargs.SQL
	twitchargs.Twitch
	ircargs.IRC
	redisargs.Redis
	nsqargs.NSQ
	jaegerargs.Jaeger
	rlargs.RateLimit
}{
	Common:    cmdargs.DefaultCommon,
	SQL:       sqlargs.DefaultSQL,
	Twitch:    twitchargs.DefaultTwitch,
	IRC:       ircargs.DefaultIRC,
	Redis:     redisargs.DefaultRedis,
	NSQ:       nsqargs.DefaultNSQ,
	Jaeger:    jaegerargs.DefaultJaeger,
	RateLimit: rlargs.DefaultRateLimit,
}

func main() {
	cmdargs.Run(&args, mainCtx)
}

//nolint:gocyclo
func mainCtx(ctx context.Context) {
	logger := ctxlog.FromContext(ctx)

	defer args.InitJaeger(ctx, "irc", args.Debug)()

	connector := args.DBConnector(ctx)
	connector = args.TraceDB(args.Debug, connector)
	db := args.OpenDB(ctx, connector)

	rdb := args.RedisClient()
	twitchAPI := args.TwitchClient()
	conn := args.IRCPool(ctx, db, twitchAPI)

	incomingPub := bnsq.NewIncomingPublisher(args.NSQAddr)

	sendSub := args.NewSendMessageSubscriber(args.Nick, 5*time.Second, func(m *bnsq.SendMessage, parent trace.SpanContext) error {
		ctx, span := trace.StartSpanWithRemoteParent(ctx, "OnSendMessage", parent)
		defer span.End()

		allowed, err := args.SendMessageAllowed(ctx, rdb, m.Origin, m.Target)
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
	})

	syncJoined := make(chan struct{}, 1)

	notifySub := args.NewNotifySubscriber(args.Nick, time.Minute, func(n *bnsq.ChannelUpdatesNotification, parent trace.SpanContext) error {
		ctx, span := trace.StartSpanWithRemoteParent(ctx, "OnNotifyChannelUpdates", parent)
		defer span.End()

		select {
		case syncJoined <- struct{}{}:
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		return nil
	})

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
