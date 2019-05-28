package main

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"os/signal"
	"time"

	"github.com/hortbot/hortbot/internal/birc"
	"github.com/hortbot/hortbot/internal/bot"
	"github.com/hortbot/hortbot/internal/ctxlog"
	"github.com/hortbot/hortbot/internal/db/models"
	"github.com/hortbot/hortbot/internal/dedupe/memory"
	"github.com/hortbot/hortbot/internal/x/errgroupx"
	"github.com/stevenroose/gonfig"
	"github.com/volatiletech/sqlboiler/queries/qm"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	_ "github.com/joho/godotenv/autoload" // Pull .env into env vars.
	_ "github.com/lib/pq"                 // For postgres
)

var config = struct {
	Debug bool

	Nick string
	Pass string

	DB string

	Admins []string

	WhitelistEnabled bool
	Whitelist        []string
}{}

func main() {
	ctx := withSignalCancel(context.Background(), os.Interrupt)

	// TODO: Replace gonfig with something else.
	if err := gonfig.Load(&config, gonfig.Conf{
		EnvPrefix: "HB_",
	}); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	// TODO: validate config

	logger := buildLogger(config.Debug)

	undoStdlog := zap.RedirectStdLog(logger)
	defer undoStdlog()

	ctx = ctxlog.WithLogger(ctx, logger)

	db, err := sql.Open("postgres", config.DB)
	if err != nil {
		logger.Fatal("error opening database connection", zap.Error(err))
	}

	channels, err := listChannels(ctx, db)
	if err != nil {
		logger.Fatal("error listing initial channels", zap.Error(err))
	}

	pc := birc.PoolConfig{
		Config: birc.Config{
			UserConfig: birc.UserConfig{
				Nick: config.Nick,
				Pass: config.Pass,
			},
			InitialChannels: channels,
			Caps:            []string{birc.TwitchCapCommands, birc.TwitchCapTags},
		},
	}

	conn := birc.NewPool(pc)

	sender := bot.SenderFuncs{
		SendMessageFunc: func(origin, target, message string) error {
			return conn.SendMessage(ctx, target, message)
		},
	}

	syncJoined := make(chan struct{}, 1)

	notifier := bot.NotifierFuncs{
		NotifyChannelUpdatesFunc: func(botName string) {
			logger.Debug("notified update to channels for bot", zap.String("botName", botName))
			select {
			case syncJoined <- struct{}{}:
			default:
			}
		},
	}

	ddp := memory.New(time.Minute, 5*time.Minute)
	defer ddp.Stop()

	bc := &bot.Config{
		DB:               db,
		Dedupe:           ddp,
		Sender:           sender,
		Notifier:         notifier,
		Admins:           config.Admins,
		WhitelistEnabled: config.WhitelistEnabled,
		Whitelist:        config.Whitelist,
	}

	b := bot.New(bc)

	g := errgroupx.FromContext(ctx)

	g.Go(func(ctx context.Context) error {
		inc := conn.Incoming()

		for {
			select {
			case <-ctx.Done():
				return ctx.Err()

			case m := <-inc:
				g.Go(func(ctx context.Context) error {
					b.Handle(ctx, config.Nick, m)
					return nil
				})
			}
		}
	})

	g.Go(func(ctx context.Context) error {
		for {
			select {
			case <-ctx.Done():
				return ctx.Err()

			case <-syncJoined:
				time.Sleep(time.Second) // The notification comes in before the transaction is complete.

			case <-time.After(time.Minute):
			}

			channels, err := listChannels(ctx, db)
			if err != nil {
				return err
			}

			if err := conn.SyncJoined(ctx, channels...); err != nil {
				return err
			}
		}
	})

	g.Go(conn.Run)

	if err := g.WaitIgnoreStop(); err != nil {
		logger.Info("exiting", zap.Error(err))
	}
}

func withSignalCancel(ctx context.Context, sig ...os.Signal) context.Context {
	ctx, cancel := context.WithCancel(ctx)
	go func() {
		c := make(chan os.Signal, 1)
		signal.Notify(c, sig...)
		defer signal.Stop(c)

		select {
		case <-c:
			cancel()
		case <-ctx.Done():
		}
	}()

	return ctx
}

func buildLogger(debug bool) *zap.Logger {
	var logConfig zap.Config

	if debug {
		logConfig = zap.NewDevelopmentConfig()
	} else {
		logConfig = zap.NewProductionConfig()
	}

	logConfig.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder

	logger, err := logConfig.Build()
	if err != nil {
		panic(err)
	}

	return logger
}

func listChannels(ctx context.Context, db *sql.DB) ([]string, error) {
	var channels []struct {
		Name string
	}

	err := models.Channels(
		qm.Select(models.ChannelColumns.Name),
		models.ChannelWhere.Active.EQ(true),
		models.ChannelWhere.BotName.EQ(config.Nick),
	).Bind(ctx, db, &channels)

	if err != nil {
		return nil, err
	}

	out := make([]string, len(channels), len(channels)+1)

	for i, c := range channels {
		out[i] = "#" + c.Name
	}

	out = append(out, "#"+config.Nick)

	return out, nil
}
