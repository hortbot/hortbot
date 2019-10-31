// Package ircflags processes IRC-related flags.
package ircflags

import (
	"context"
	"database/sql"
	"time"

	"github.com/hortbot/hortbot/internal/birc"
	"github.com/hortbot/hortbot/internal/db/modelsx"
	"github.com/hortbot/hortbot/internal/db/redis"
	"github.com/hortbot/hortbot/internal/pkg/apis/twitch"
	"github.com/hortbot/hortbot/internal/pkg/ctxlog"
	"github.com/hortbot/hortbot/internal/pkg/twitchx"
	"go.uber.org/zap"
)

type IRC struct {
	Nick         string        `long:"nick" env:"HB_NICK" description:"IRC nick" required:"true"`
	Pass         string        `long:"pass" env:"HB_PASS" description:"IRC pass" required:"true"`
	PingInterval time.Duration `long:"ping-interval" env:"HB_PING_INTERVAL" description:"How often to ping the IRC server"`
	PingDeadline time.Duration `long:"ping-deadline" env:"HB_PING_DEADLINE" description:"How long to wait for a PONG before disconnecting"`
	NoToken      bool          `long:"no-token" env:"HB_NO_TOKEN" description:"Don't use a token from the database"`

	RateLimitSlow   int           `long:"rate-limit-slow" env:"HB_RATE_LIMIT_RATE" description:"Message allowed per rate limit period (slow)"`
	RateLimitFast   int           `long:"rate-limit-fast" env:"HB_RATE_LIMIT_RATE" description:"Message allowed per rate limit period (fast)"`
	RateLimitPeriod time.Duration `long:"rate-limit-period" env:"HB_RATE_LIMIT_PERIOD" description:"Rate limit period"`
}

var DefaultIRC = IRC{
	PingInterval:    5 * time.Minute,
	PingDeadline:    5 * time.Second,
	RateLimitSlow:   15,
	RateLimitFast:   80,
	RateLimitPeriod: 30 * time.Second,
}

func (args *IRC) IRCPool(ctx context.Context, db *sql.DB, tw twitch.API) *birc.Pool {
	logger := ctxlog.FromContext(ctx)

	channels, err := modelsx.ListActiveChannels(ctx, db, args.Nick)
	if err != nil {
		logger.Fatal("error listing initial channels", zap.Error(err))
	}

	nick := args.Nick
	pass := args.Pass

	if !args.NoToken {
		token, err := twitchx.FindBotToken(ctx, db, tw, nick)
		if err != nil {
			logger.Fatal("error querying for bot token", zap.Error(err))
		}
		if token != nil {
			logger.Debug("using token from database")
			pass = "oauth:" + token.AccessToken
		}
	}

	pc := birc.PoolConfig{
		Config: birc.Config{
			UserConfig: birc.UserConfig{
				Nick: nick,
				Pass: pass,
			},
			InitialChannels: channels,
			Caps:            []string{birc.TwitchCapCommands, birc.TwitchCapTags},
			PingInterval:    args.PingInterval,
			PingDeadline:    args.PingDeadline,
		},
	}

	return birc.NewPool(pc)
}

func (args *IRC) SendMessageAllowed(ctx context.Context, rdb *redis.DB, origin, target string) (bool, error) {
	return rdb.SendMessageAllowed(ctx, origin, target, args.RateLimitSlow, args.RateLimitFast, args.RateLimitPeriod)
}
