package main

import (
	"context"

	"github.com/hortbot/hortbot/internal/cmdargs"
	"github.com/hortbot/hortbot/internal/cmdargs/jaegerargs"
	"github.com/hortbot/hortbot/internal/cmdargs/redisargs"
	"github.com/hortbot/hortbot/internal/cmdargs/sqlargs"
	"github.com/hortbot/hortbot/internal/cmdargs/twitchargs"
	"github.com/hortbot/hortbot/internal/cmdargs/webargs"
	"github.com/hortbot/hortbot/internal/pkg/ctxlog"
	"go.uber.org/zap"
)

var args = struct {
	cmdargs.Common
	sqlargs.SQL
	twitchargs.Twitch
	redisargs.Redis
	webargs.Web
	jaegerargs.Jaeger
}{
	Common: cmdargs.DefaultCommon,
	SQL:    sqlargs.DefaultSQL,
	Twitch: twitchargs.DefaultTwitch,
	Redis:  redisargs.DefaultRedis,
	Web:    webargs.DefaultWeb,
	Jaeger: jaegerargs.DefaultJaeger,
}

func main() {
	cmdargs.Run(&args, mainCtx)
}

func mainCtx(ctx context.Context) {
	logger := ctxlog.FromContext(ctx)

	defer args.InitJaeger(ctx, "web", args.Debug)()

	connector := args.DBConnector(ctx)
	connector = args.TraceDB(args.Debug, connector)
	db := args.OpenDB(ctx, connector)

	rdb := args.RedisClient()
	tw := args.TwitchClient()
	a := args.WebApp(args.Debug, rdb, db, tw)

	err := a.Run(ctx)
	logger.Info("exiting", zap.Error(err))
}
