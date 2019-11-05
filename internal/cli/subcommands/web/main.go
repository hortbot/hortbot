package web

import (
	"context"

	"github.com/hortbot/hortbot/internal/cli"
	"github.com/hortbot/hortbot/internal/cli/flags/jaegerflags"
	"github.com/hortbot/hortbot/internal/cli/flags/redisflags"
	"github.com/hortbot/hortbot/internal/cli/flags/sqlflags"
	"github.com/hortbot/hortbot/internal/cli/flags/twitchflags"
	"github.com/hortbot/hortbot/internal/cli/flags/webflags"
	"github.com/hortbot/hortbot/internal/pkg/ctxlog"
	"go.uber.org/zap"
)

type cmd struct {
	cli.Common
	SQL    sqlflags.SQL
	Twitch twitchflags.Twitch
	Redis  redisflags.Redis
	Web    webflags.Web
	Jaeger jaegerflags.Jaeger
}

func Run(args []string) {
	cli.Run("web", args, &cmd{
		Common: cli.DefaultCommon,
		SQL:    sqlflags.DefaultSQL,
		Twitch: twitchflags.DefaultTwitch,
		Redis:  redisflags.DefaultRedis,
		Web:    webflags.DefaultWeb,
		Jaeger: jaegerflags.DefaultJaeger,
	})
}

func (cmd *cmd) Main(ctx context.Context, _ []string) {
	logger := ctxlog.FromContext(ctx)

	defer cmd.Jaeger.Init(ctx, "web", cmd.Debug)()

	connector := cmd.SQL.Connector(ctx)
	connector = cmd.Jaeger.TraceDB(cmd.Debug, connector)
	db := cmd.SQL.Open(ctx, connector)

	rdb := cmd.Redis.Client()
	tw := cmd.Twitch.Client()
	a := cmd.Web.New(cmd.Debug, rdb, db, tw)

	err := a.Run(ctx)
	logger.Info("exiting", zap.Error(err))
}
