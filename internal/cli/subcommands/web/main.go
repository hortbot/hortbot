// Package web implements the main command for the web service.
package web

import (
	"context"

	"github.com/hortbot/hortbot/internal/cli"
	"github.com/hortbot/hortbot/internal/cli/flags/httpflags"
	"github.com/hortbot/hortbot/internal/cli/flags/nsqflags"
	"github.com/hortbot/hortbot/internal/cli/flags/promflags"
	"github.com/hortbot/hortbot/internal/cli/flags/redisflags"
	"github.com/hortbot/hortbot/internal/cli/flags/sqlflags"
	"github.com/hortbot/hortbot/internal/cli/flags/twitchflags"
	"github.com/hortbot/hortbot/internal/cli/flags/webflags"
	"github.com/hortbot/hortbot/internal/pkg/errgroupx"
	"github.com/zikaeroh/ctxlog"
	"go.uber.org/zap"
)

type cmd struct {
	cli.Common
	SQL        sqlflags.SQL
	Twitch     twitchflags.Twitch
	Redis      redisflags.Redis
	Web        webflags.Web
	NSQ        nsqflags.NSQ
	Prometheus promflags.Prometheus
	HTTP       httpflags.HTTP
}

// Command returns a fresh web command.
func Command() cli.Command {
	return &cmd{
		Common:     cli.Default,
		SQL:        sqlflags.Default,
		Twitch:     twitchflags.Default,
		Redis:      redisflags.Default,
		Web:        webflags.Default,
		NSQ:        nsqflags.Default,
		Prometheus: promflags.Default,
		HTTP:       httpflags.Default,
	}
}

func (*cmd) Name() string {
	return "web"
}

func (c *cmd) Main(ctx context.Context, _ []string) {
	c.Prometheus.Run(ctx)

	driverName := c.SQL.DriverName()
	db := c.SQL.Open(ctx, driverName)

	rdb := c.Redis.Client()
	tw := c.Twitch.Client(c.HTTP.Client())
	a := c.Web.New(c.Debug, rdb, db, tw)
	eventsubNotifier := c.NSQ.NewEventsubNotifyPublisher()
	a.EventsubUpdateNotifier = eventsubNotifier

	g := errgroupx.FromContext(ctx)
	g.Go(eventsubNotifier.Run)
	g.Go(a.Run)

	err := g.WaitIgnoreStop()
	ctxlog.Info(ctx, "exiting", zap.Error(err))
}
