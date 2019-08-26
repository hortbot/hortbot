package main

import (
	"database/sql"
	"log"
	"os"
	"time"

	"github.com/go-redis/redis/v7"
	"github.com/hortbot/hortbot/internal/db/migrations"
	"github.com/hortbot/hortbot/internal/pkg/apis/twitch"
	"github.com/hortbot/hortbot/internal/pkg/ctxlog"
	"github.com/hortbot/hortbot/internal/pkg/rdb"
	"github.com/hortbot/hortbot/internal/web"
	"github.com/jessevdk/go-flags"
	"github.com/posener/ctxutil"
	"go.uber.org/zap"

	_ "github.com/joho/godotenv/autoload" // Pull .env into env vars.
	_ "github.com/lib/pq"                 // For postgres
)

var args = struct {
	DB    string `long:"db" env:"HB_DB" description:"PostgresSQL connection string" required:"true"`
	Redis string `long:"redis" env:"HB_REDIS" description:"Redis address" required:"true"`

	TwitchClientID     string `long:"twitch-client-id" env:"HB_TWITCH_CLIENT_ID" description:"Twitch OAuth client ID" required:"true"`
	TwitchClientSecret string `long:"twitch-client-secret" env:"HB_TWITCH_CLIENT_SECRET" description:"Twitch OAuth client secret" required:"true"`
	TwitchRedirectURL  string `long:"twitch-redirect-url" env:"HB_TWITCH_REDIRECT_URL" description:"Twitch OAuth redirect URL" required:"true"`

	WebAddr string `long:"web-addr" env:"HB_WEB_ADDR" description:"Server address for the web server"`

	Debug     bool `long:"debug" env:"HB_DEBUG" description:"Enables debug mode and the debug log level"`
	MigrateUp bool `long:"migrate-up" env:"HB_MIGRATE_UP" description:"Migrates the postgres database up"`
}{
	WebAddr: ":5000",
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

	db, err := sql.Open("postgres", args.DB)
	if err != nil {
		logger.Fatal("error opening database connection", zap.Error(err))
	}

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

	rClient := redis.NewClient(&redis.Options{
		Addr: args.Redis,
	})
	defer rClient.Close()

	webRDB, err := rdb.New(rClient, rdb.KeyPrefix("web"))
	if err != nil {
		logger.Fatal("error creating RDB", zap.Error(err))
	}

	twitchAPI := twitch.New(args.TwitchClientID, args.TwitchClientSecret, args.TwitchRedirectURL)

	a := web.App{
		Addr:   args.WebAddr,
		RDB:    webRDB,
		DB:     db,
		Twitch: twitchAPI,
	}

	err = a.Run(ctx)
	logger.Info("exiting", zap.Error(err))
}
