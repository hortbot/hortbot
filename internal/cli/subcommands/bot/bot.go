// Package bot implements the main command for the bot service.
package bot

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"time"

	"github.com/hortbot/hortbot/internal/bnsq"
	"github.com/hortbot/hortbot/internal/bot"
	"github.com/hortbot/hortbot/internal/bot/eventsubtobot"
	"github.com/hortbot/hortbot/internal/cli"
	"github.com/hortbot/hortbot/internal/cli/flags/botflags"
	"github.com/hortbot/hortbot/internal/cli/flags/httpflags"
	"github.com/hortbot/hortbot/internal/cli/flags/nsqflags"
	"github.com/hortbot/hortbot/internal/cli/flags/promflags"
	"github.com/hortbot/hortbot/internal/cli/flags/redisflags"
	"github.com/hortbot/hortbot/internal/cli/flags/sqlflags"
	"github.com/hortbot/hortbot/internal/cli/flags/twitchflags"
	"github.com/hortbot/hortbot/internal/db/modelsx"
	"github.com/hortbot/hortbot/internal/pkg/errgroupx"
	"github.com/hortbot/hortbot/internal/pkg/wqueue"
	"github.com/zikaeroh/ctxlog"
	"go.uber.org/zap"
)

type cmd struct {
	cli.Common
	SQL        sqlflags.SQL
	Twitch     twitchflags.Twitch
	Redis      redisflags.Redis
	Bot        botflags.Bot
	NSQ        nsqflags.NSQ
	Prometheus promflags.Prometheus
	HTTP       httpflags.HTTP
}

// Command returns a fresh bot command.
func Command() cli.Command {
	return &cmd{
		Common:     cli.Default,
		SQL:        sqlflags.Default,
		Twitch:     twitchflags.Default,
		Redis:      redisflags.Default,
		Bot:        botflags.Default,
		NSQ:        nsqflags.Default,
		Prometheus: promflags.Default,
		HTTP:       httpflags.Default,
	}
}

func (*cmd) Name() string {
	return "bot"
}

func (c *cmd) Main(ctx context.Context, _ []string) {
	c.Prometheus.Run(ctx)

	httpClient := c.HTTP.Client()
	untrustedClient := c.HTTP.UntrustedClient(ctx)
	driverName := c.SQL.DriverName()
	db := c.SQL.Open(ctx, driverName)
	rdb := c.Redis.Client()
	twitchAPI := c.Twitch.Client(httpClient)
	eventsubNotifier := c.NSQ.NewEventsubNotifyPublisher()

	b := c.Bot.New(ctx, db, rdb, eventsubNotifier, twitchAPI, httpClient, untrustedClient)
	defer b.Stop()

	g := errgroupx.FromContext(ctx)

	workers := c.Bot.Workers
	if workers < 1 {
		workers = runtime.GOMAXPROCS(0)
	}

	// TODO: pass the queue down to the bot to use internally
	queue := wqueue.NewQueue[string](10 * workers)
	for range workers {
		g.Go(queue.Worker)
	}

	put := func(subCtx context.Context, metadata *bnsq.Metadata, mm bot.Message) error {
		key := mm.BroadcasterLogin()
		return queue.Put(subCtx, key, func(attach wqueue.Attacher) {
			ctx, cancel := attach(ctx)
			defer cancel()

			ctx = metadata.With(ctx)
			b.Handle(ctx, mm)
		})
	}

	// For now, the bot needs the login name of the bot for the "origin".
	// Periodically get that mapping and use it when constructing messages.
	// TODO: remove concept of "origin" once IRC is gone?
	var mu sync.Mutex
	var originMap map[int64]string
	var originMapTimestamp time.Time
	getOriginMap := func(ctx context.Context) (map[int64]string, error) {
		mu.Lock()
		defer mu.Unlock()

		if originMap != nil || time.Since(originMapTimestamp) < 5*time.Minute {
			return originMap, nil
		}

		var err error
		_, originMap, err = modelsx.GetBots(ctx, db)
		if err != nil {
			return nil, fmt.Errorf("get bots: %w", err)
		}
		return originMap, nil
	}

	eventsubSub := c.NSQ.NewIncomingWebsocketMessageSubscriber(15*time.Second, func(i *bnsq.IncomingWebsocketMessage, metadata *bnsq.Metadata) error {
		originMap, err := getOriginMap(ctx)
		if err != nil {
			return err
		}

		mm := eventsubtobot.ToMessage(originMap, i.Message)
		return put(ctx, metadata, mm)
	})

	g.Go(eventsubNotifier.Run)
	g.Go(eventsubSub.Run)

	if err := g.WaitIgnoreStop(); err != nil {
		ctxlog.Info(ctx, "exiting", zap.Error(err))
	}
}
