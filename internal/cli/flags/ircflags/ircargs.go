// Package ircflags processes IRC-related flags.
package ircflags

import (
	"context"
	"database/sql"
	"time"

	"github.com/hortbot/hortbot/internal/birc"
	"github.com/hortbot/hortbot/internal/db/modelsx"
	"github.com/hortbot/hortbot/internal/pkg/apiclient/twitch"
	"github.com/hortbot/hortbot/internal/pkg/twitchx"
	"github.com/zikaeroh/ctxlog"
	"go.uber.org/zap"
)

// IRC contains IRC client flags.
type IRC struct {
	Nick         string        `long:"irc-nick" env:"HB_IRC_NICK" description:"IRC nick" required:"true"`
	Pass         string        `long:"irc-pass" env:"HB_IRC_PASS" description:"IRC pass" required:"true"`
	Addr         string        `long:"irc-addr" env:"HB_IRC_ADDR" description:"IRC address"`
	PingInterval time.Duration `long:"irc-ping-interval" env:"HB_IRC_PING_INTERVAL" description:"How often to ping the IRC server"`
	PingDeadline time.Duration `long:"irc-ping-deadline" env:"HB_IRC_PING_DEADLINE" description:"How long to wait for a PONG before disconnecting"`
	Token        bool          `long:"irc-token" env:"HB_IRC_TOKEN" description:"Use a token from the database if available"`

	PriorityChannels []string `long:"irc-priority-channels" env:"HB_IRC_PRIORITY_CHANNELS" env-delim:"," description:"An ordered list of channels to prioritize joining"`
}

// Default contains the default flags. Make a copy of this, do not reuse.
var Default = IRC{
	Addr:         birc.DefaultDialer.Addr,
	PingInterval: 5 * time.Minute,
	PingDeadline: 5 * time.Second,
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
			Dialer: &birc.Dialer{
				Addr: args.Addr,
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
