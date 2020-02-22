// Package ircflags processes IRC-related flags.
package ircflags

import (
	"context"
	"database/sql"
	"time"

	"github.com/hortbot/hortbot/internal/birc"
	"github.com/hortbot/hortbot/internal/db/modelsx"
	"github.com/hortbot/hortbot/internal/db/redis"
	"github.com/hortbot/hortbot/internal/pkg/apiclient/twitch"
	"github.com/hortbot/hortbot/internal/pkg/ctxlog"
	"github.com/hortbot/hortbot/internal/pkg/twitchx"
	"go.uber.org/zap"
)

// IRC contains IRC client flags.
type IRC struct {
	Nick         string        `long:"irc-nick" env:"HB_IRC_NICK" description:"IRC nick" required:"true"`
	Pass         string        `long:"irc-pass" env:"HB_IRC_PASS" description:"IRC pass" required:"true"`
	PingInterval time.Duration `long:"irc-ping-interval" env:"HB_IRC_PING_INTERVAL" description:"How often to ping the IRC server"`
	PingDeadline time.Duration `long:"irc-ping-deadline" env:"HB_IRC_PING_DEADLINE" description:"How long to wait for a PONG before disconnecting"`
	Token        bool          `long:"irc-token" env:"HB_IRC_TOKEN" description:"Use a token from the database if available"`

	RateLimitSlow   int           `long:"irc-rate-limit-slow" env:"HB_IRC_RATE_LIMIT_SLOW" description:"Message allowed per rate limit period (slow)"`
	RateLimitFast   int           `long:"irc-rate-limit-fast" env:"HB_IRC_RATE_LIMIT_FAST" description:"Message allowed per rate limit period (fast)"`
	RateLimitPeriod time.Duration `long:"irc-rate-limit-period" env:"HB_IRC_RATE_LIMIT_PERIOD" description:"Rate limit period"`

	PriorityChannels []string `long:"irc-priority-channels" env:"HB_IRC_PRIORITY_CHANNELS" description:"An ordered list of channels to prioritize joining"`
}

// Default contains the default flags. Make a copy of this, do not reuse.
var Default = IRC{
	PingInterval:    5 * time.Minute,
	PingDeadline:    5 * time.Second,
	RateLimitSlow:   15,
	RateLimitFast:   80,
	RateLimitPeriod: 30 * time.Second,
}

// Pool creates a new IRC pool from the configured flags and dependency.
func (args *IRC) Pool(ctx context.Context, db *sql.DB, tw twitch.API) *birc.Pool {
	channels, err := modelsx.ListActiveChannels(ctx, db, args.Nick)
	if err != nil {
		ctxlog.Fatal(ctx, "error listing initial channels", zap.Error(err))
	}

	nick := args.Nick
	pass := args.Pass

	if args.Token {
		token, err := twitchx.FindBotToken(ctx, db, tw, nick)
		if err != nil {
			ctxlog.Fatal(ctx, "error querying for bot token", zap.Error(err))
		}
		if token != nil {
			ctxlog.Debug(ctx, "using token from database")
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
		PriorityChannels: args.PriorityChannels,
	}

	return birc.NewPool(pc)
}

// SendMessageAllowed checks if sending a message is allowed under the current
// rate limit
func (args *IRC) SendMessageAllowed(ctx context.Context, rdb *redis.DB, origin, target string) (bool, error) {
	return rdb.SendMessageAllowed(ctx, origin, target, args.RateLimitSlow, args.RateLimitFast, args.RateLimitPeriod)
}
