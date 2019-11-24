package web

import (
	"context"

	"github.com/hortbot/hortbot/internal/cli"
	"github.com/hortbot/hortbot/internal/cli/flags/jaegerflags"
	"github.com/hortbot/hortbot/internal/cli/flags/promflags"
	"github.com/hortbot/hortbot/internal/cli/flags/redisflags"
	"github.com/hortbot/hortbot/internal/cli/flags/sqlflags"
	"github.com/hortbot/hortbot/internal/cli/flags/twitchflags"
	"github.com/hortbot/hortbot/internal/cli/flags/webflags"
	"github.com/hortbot/hortbot/internal/pkg/ctxlog"
	"go.uber.org/zap"
)

const Name = "web"

type cmd struct {
	cli.Common
	SQL        sqlflags.SQL
	Twitch     twitchflags.Twitch
	Redis      redisflags.Redis
	Web        webflags.Web
	Jaeger     jaegerflags.Jaeger
	Prometheus promflags.Prometheus
}

func Run(args []string) {
	cli.Run(Name, args, &cmd{
		Common:     cli.DefaultCommon,
		SQL:        sqlflags.DefaultSQL,
		Twitch:     twitchflags.DefaultTwitch,
		Redis:      redisflags.DefaultRedis,
		Web:        webflags.DefaultWeb,
		Jaeger:     jaegerflags.DefaultJaeger,
		Prometheus: promflags.Default,
	})
}

func (cmd *cmd) Main(ctx context.Context, _ []string) {
	defer cmd.Jaeger.Init(ctx, Name, cmd.Debug)()
	cmd.Prometheus.Run(ctx)

	connector := cmd.SQL.Connector(ctx)
	connector = cmd.Jaeger.TraceDB(cmd.Debug, connector)
	db := cmd.SQL.Open(ctx, connector)

	rdb := cmd.Redis.Client()
	tw := cmd.Twitch.Client()
	a := cmd.Web.New(cmd.Debug, rdb, db, tw)

	err := a.Run(ctx)
	ctxlog.Info(ctx, "exiting", zap.Error(err))
}