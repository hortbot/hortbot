package irc

import (
	"context"
	"errors"
	"time"

	"github.com/hortbot/hortbot/internal/bnsq"
	"github.com/hortbot/hortbot/internal/cli"
	"github.com/hortbot/hortbot/internal/cli/flags/httpflags"
	"github.com/hortbot/hortbot/internal/cli/flags/ircflags"
	"github.com/hortbot/hortbot/internal/cli/flags/jaegerflags"
	"github.com/hortbot/hortbot/internal/cli/flags/nsqflags"
	"github.com/hortbot/hortbot/internal/cli/flags/promflags"
	"github.com/hortbot/hortbot/internal/cli/flags/redisflags"
	"github.com/hortbot/hortbot/internal/cli/flags/sqlflags"
	"github.com/hortbot/hortbot/internal/cli/flags/twitchflags"
	"github.com/hortbot/hortbot/internal/db/modelsx"
	"github.com/hortbot/hortbot/internal/pkg/ctxlog"
	"github.com/hortbot/hortbot/internal/pkg/errgroupx"
	"go.opencensus.io/trace"
	"go.uber.org/zap"
)

const Name = "irc"

type cmd struct {
	cli.Common
	SQL        sqlflags.SQL
	Twitch     twitchflags.Twitch
	IRC        ircflags.IRC
	Redis      redisflags.Redis
	NSQ        nsqflags.NSQ
	Jaeger     jaegerflags.Jaeger
	Prometheus promflags.Prometheus
	HTTP       httpflags.HTTP
}

func Run(args []string) {
	cli.Run(Name, args, &cmd{
		Common:     cli.DefaultCommon,
		SQL:        sqlflags.DefaultSQL,
		Twitch:     twitchflags.DefaultTwitch,
		IRC:        ircflags.DefaultIRC,
		Redis:      redisflags.DefaultRedis,
		NSQ:        nsqflags.DefaultNSQ,
		Jaeger:     jaegerflags.DefaultJaeger,
		Prometheus: promflags.Default,
		HTTP:       httpflags.DefaultHTTP,
	})
}

//nolint:gocyclo
func (cmd *cmd) Main(ctx context.Context, _ []string) {
	defer cmd.Jaeger.Init(ctx, Name, cmd.Debug)()
	cmd.Prometheus.Run(ctx)

	driverName := cmd.SQL.DriverName()
	driverName = cmd.Jaeger.DriverName(ctx, driverName, cmd.Debug)
	db := cmd.SQL.Open(ctx, driverName)

	rdb := cmd.Redis.Client()
	twitchAPI := cmd.Twitch.Client(cmd.HTTP.Client())
	conn := cmd.IRC.Pool(ctx, db, twitchAPI)

	incomingPub := cmd.NSQ.NewIncomingPublisher()

	sendSub := cmd.NSQ.NewSendMessageSubscriber(cmd.IRC.Nick, 15*time.Second, func(m *bnsq.SendMessage, metadata *bnsq.Metadata) error {
		ctx := metadata.With(ctx)
		ctx, span := trace.StartSpanWithRemoteParent(ctx, "OnSendMessage", metadata.ParentSpan())
		defer span.End()

		allowed, err := cmd.IRC.SendMessageAllowed(ctx, rdb, m.Origin, m.Target)
		if err != nil {
			ctxlog.Error(ctx, "error checking rate limit", zap.Error(err))
			return err
		}

		if !allowed {
			ctxlog.Error(ctx, "rate limited, requeueing")
			return errors.New("rate limited")
		}

		if err := conn.SendMessage(ctx, m.Target, m.Message); err != nil {
			ctxlog.Error(ctx, "error sending message", zap.Error(err))
			return err
		}

		return nil
	})

	syncJoined := make(chan struct{}, 1)

	notifySub := cmd.NSQ.NewNotifySubscriber(cmd.IRC.Nick, time.Minute, func(n *bnsq.ChannelUpdatesNotification, metadata *bnsq.Metadata) error {
		ctx := metadata.With(ctx)
		ctx, span := trace.StartSpanWithRemoteParent(ctx, "OnNotifyChannelUpdates", metadata.ParentSpan())
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

				if err := incomingPub.Publish(ctx, cmd.IRC.Nick, m); err != nil {
					ctxlog.Error(ctx, "error publishing incoming message", zap.Error(err))
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

			channels, err := modelsx.ListActiveChannels(ctx, db, cmd.IRC.Nick)
			if err != nil {
				return err
			}

			if err := conn.SyncJoined(ctx, channels...); err != nil {
				return err
			}
		}
	})

	if err := g.WaitIgnoreStop(); err != nil {
		ctxlog.Info(ctx, "exiting", zap.Error(err))
	}
}
