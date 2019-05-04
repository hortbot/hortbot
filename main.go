package main

import (
	"context"
	"os"
	"os/signal"
	"time"

	"github.com/hortbot/hortbot/internal/birc"
	"github.com/hortbot/hortbot/internal/ctxlog"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	_ "github.com/joho/godotenv/autoload" // Pull .env into env vars.
)

func lookupEnv(key string) string {
	v, _ := os.LookupEnv(key)
	return v
}

func main() {
	ctx := withSignalCancel(context.Background(), os.Interrupt)

	logConfig := zap.NewDevelopmentConfig()
	logConfig.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder

	logger, err := logConfig.Build()
	if err != nil {
		panic(err)
	}

	undoStdlog := zap.RedirectStdLog(logger)
	defer undoStdlog()

	ctx = ctxlog.WithLogger(ctx, logger)

	conn := birc.NewPool(birc.PoolConfig{
		Config: birc.Config{
			UserConfig: birc.UserConfig{
				Nick: lookupEnv("HB_NICK"),
				Pass: lookupEnv("HB_PASS"),
			},
			InitialChannels: []string{
				"#coestar",
				"#zikaeroh",
				"#erei",
				// "#guude",
				// "#last_grey_wolf",
				// "#yolopanther",
				// "#hortbot",
				// "#botzik",
				// "#pause",
				// "#flackblag",
				// "#northernlion",
			},
			Caps: []string{birc.TwitchCapCommands, birc.TwitchCapTags},
		},
		// MaxChannelsPerSubConn: 1,
	})

	go func() {
		for m := range conn.Incoming() {
			logger.Info(m.Raw)
		}
	}()

	go func() {
		defer func() {
			logger.Info("after sync", zap.Strings("joined", conn.Joined()))
		}()

		select {
		case <-time.After(5 * time.Second):
			logger.Info("before sync", zap.Strings("joined", conn.Joined()))
		case <-ctx.Done():
			return
		}

		select {
		case <-time.After(30 * time.Second):
			if err := conn.SyncJoined(ctx, "#zikaeroh"); err != nil {
				logger.Error("error syncing", zap.Error(err))
				return
			}
		case <-ctx.Done():
			return
		}
	}()

	if err := conn.Run(ctx); err != nil {
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
