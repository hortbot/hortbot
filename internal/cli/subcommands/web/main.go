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
	sqlflags.SQL
	twitchflags.Twitch
	redisflags.Redis
	webflags.Web
	jaegerflags.Jaeger
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

	defer cmd.InitJaeger(ctx, "web", cmd.Debug)()

	connector := cmd.DBConnector(ctx)
	connector = cmd.TraceDB(cmd.Debug, connector)
	db := cmd.OpenDB(ctx, connector)

	rdb := cmd.RedisClient()
	tw := cmd.TwitchClient()
	a := cmd.WebApp(cmd.Debug, rdb, db, tw)

	err := a.Run(ctx)
	logger.Info("exiting", zap.Error(err))
}
