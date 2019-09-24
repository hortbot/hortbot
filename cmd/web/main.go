package main

import (
	"context"
	"database/sql"
	"time"

	"contrib.go.opencensus.io/integrations/ocsql"
	goredis "github.com/go-redis/redis/v7"
	"github.com/hortbot/hortbot/internal/cmdargs"
	"github.com/hortbot/hortbot/internal/db/migrations"
	"github.com/hortbot/hortbot/internal/db/redis"
	"github.com/hortbot/hortbot/internal/pkg/apis/twitch"
	"github.com/hortbot/hortbot/internal/pkg/ctxlog"
	"github.com/hortbot/hortbot/internal/pkg/tracing"
	"github.com/hortbot/hortbot/internal/web"
	"github.com/lib/pq"
	"go.uber.org/zap"

	_ "github.com/joho/godotenv/autoload" // Pull .env into env vars.
)

var args = struct {
	cmdargs.Common
	cmdargs.SQL
	cmdargs.Redis
	cmdargs.Twitch
	cmdargs.Web
	cmdargs.Jaeger
}{
	Common: cmdargs.DefaultCommon,
	SQL:    cmdargs.DefaultSQL,
	Redis:  cmdargs.DefaultRedis,
	Twitch: cmdargs.DefaultTwitch,
	Web:    cmdargs.DefaultWeb,
	Jaeger: cmdargs.DefaultJaeger,
}

func main() {
	cmdargs.Run(&args, mainCtx)
}

func mainCtx(ctx context.Context) {
	logger := ctxlog.FromContext(ctx)

	if args.JaegerAgent != "" {
		flush, err := tracing.Init("web", args.JaegerAgent, args.Debug)
		if err != nil {
			logger.Fatal("error initializing tracing", zap.Error(err))
		}
		defer flush()
	}

	connector, err := pq.NewConnector(args.DB)
	if err != nil {
		logger.Fatal("error creating postgres connector", zap.Error(err))
	}

	db := sql.OpenDB(ocsql.WrapConnector(connector, ocsql.WithAllTraceOptions(), ocsql.WithQueryParams(args.Debug)))

	for i := 0; i < 5; i++ {
		if err := db.Ping(); err == nil {
			break
		}

		time.Sleep(100 * time.Millisecond)
	}

	if args.MigrateUp {
		if err := migrations.Up(args.DB, nil); err != nil {
			logger.Fatal("error migrating database", zap.Error(err))
		}
	}

	rClient := goredis.NewClient(&goredis.Options{
		Addr: args.RedisAddr,
	})
	defer rClient.Close()

	rdb := redis.New(rClient)

	twitchAPI := twitch.New(args.TwitchClientID, args.TwitchClientSecret, args.TwitchRedirectURL)

	a := web.App{
		Addr:       args.WebAddr,
		SessionKey: []byte(args.WebSessionKey),
		Brand:      args.WebBrand,
		BrandMap:   args.WebBrandMap,
		Debug:      args.Debug,
		Redis:      rdb,
		DB:         db,
		Twitch:     twitchAPI,
	}

	err = a.Run(ctx)
	logger.Info("exiting", zap.Error(err))
}
