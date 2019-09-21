package main

import (
	"database/sql"
	"log"
	"os"
	"time"

	"contrib.go.opencensus.io/integrations/ocsql"
	goredis "github.com/go-redis/redis/v7"
	"github.com/hortbot/hortbot/internal/db/migrations"
	"github.com/hortbot/hortbot/internal/db/redis"
	"github.com/hortbot/hortbot/internal/pkg/apis/twitch"
	"github.com/hortbot/hortbot/internal/pkg/ctxlog"
	"github.com/hortbot/hortbot/internal/pkg/tracing"
	"github.com/hortbot/hortbot/internal/web"
	"github.com/jessevdk/go-flags"
	"github.com/lib/pq"
	"github.com/posener/ctxutil"
	"go.uber.org/zap"

	_ "github.com/joho/godotenv/autoload" // Pull .env into env vars.
)

var args = struct {
	DB    string `long:"db" env:"HB_DB" description:"PostgresSQL connection string" required:"true"`
	Redis string `long:"redis" env:"HB_REDIS" description:"Redis address" required:"true"`

	TwitchClientID     string `long:"twitch-client-id" env:"HB_TWITCH_CLIENT_ID" description:"Twitch OAuth client ID" required:"true"`
	TwitchClientSecret string `long:"twitch-client-secret" env:"HB_TWITCH_CLIENT_SECRET" description:"Twitch OAuth client secret" required:"true"`
	TwitchRedirectURL  string `long:"twitch-redirect-url" env:"HB_TWITCH_REDIRECT_URL" description:"Twitch OAuth redirect URL" required:"true"`

	WebAddr       string            `long:"web-addr" env:"HB_WEB_ADDR" description:"Server address for the web server"`
	WebSessionKey string            `long:"web-session-key" env:"HB_WEB_SESSION_KEY" description:"Session cookie auth key"`
	WebBrand      string            `long:"web-brand" env:"HB_WEB_BRAND" description:"Web server default branding"`
	WebBrandMap   map[string]string `long:"web-brand-map" env:"HB_WEB_BRAND_MAP" env-delim:"," description:"Web server brand mapping from domains to branding (ex: 'example.com:SomeBot,other.net:WhoAmI')"`

	Debug     bool `long:"debug" env:"HB_DEBUG" description:"Enables debug mode and the debug log level"`
	MigrateUp bool `long:"migrate-up" env:"HB_MIGRATE_UP" description:"Migrates the postgres database up"`

	JaegerAgent string `long:"jaeger-agent" env:"HB_JAEGER_AGENT" description:"jaeger agent address"`
}{
	WebAddr:       ":5000",
	WebSessionKey: "this-is-insecure-do-not-use-this",
	WebBrand:      "HortBot",
}

func main() {
	ctx := ctxutil.Interrupt()

	if _, err := flags.Parse(&args); err != nil {
		if !flags.WroteHelp(err) {
			log.Fatalln(err)
		}
		os.Exit(1)
	}

	logger := ctxlog.New(args.Debug)
	defer zap.RedirectStdLog(logger)()
	ctx = ctxlog.WithLogger(ctx, logger)

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
		Addr: args.Redis,
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
