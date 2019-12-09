// Package botflags processes bot-related flags.
package botflags

import (
	"context"
	"database/sql"
	"runtime"

	"github.com/hortbot/hortbot/internal/bot"
	"github.com/hortbot/hortbot/internal/db/redis"
	"github.com/hortbot/hortbot/internal/pkg/apis/extralife"
	"github.com/hortbot/hortbot/internal/pkg/apis/lastfm"
	"github.com/hortbot/hortbot/internal/pkg/apis/steam"
	"github.com/hortbot/hortbot/internal/pkg/apis/tinyurl"
	"github.com/hortbot/hortbot/internal/pkg/apis/twitch"
	"github.com/hortbot/hortbot/internal/pkg/apis/xkcd"
	"github.com/hortbot/hortbot/internal/pkg/apis/youtube"
	"github.com/hortbot/hortbot/internal/pkg/ctxlog"
	"go.uber.org/zap"
)

type Bot struct {
	Admins []string `long:"bot-admin" env:"HB_BOT_ADMINS" env-delim:"," description:"Bot admins"`

	WhitelistEnabled bool     `long:"bot-whitelist-enabled" env:"HB_BOT_WHITELIST_ENABLED" description:"Enable the user whitelist"`
	Whitelist        []string `long:"bot-whitelist" env:"HB_BOT_WHITELIST" env-delim:"," description:"User whitelist"`

	BulletMap       map[string]string `long:"bot-bullet-map" env:"HB_BOT_BULLET_MAP" env-delim:"," description:"Mapping from bot name to default bullet"`
	DefaultCooldown int               `long:"bot-default-cooldown" env:"HB_BOT_DEFAULT_COOLDOWN" description:"default command cooldown"`

	WebAddr    string            `long:"bot-web-addr" env:"HB_BOT_WEB_ADDR" description:"Default address for the bot website"`
	WebAddrMap map[string]string `long:"bot-web-addr-map" env:"HB_BOT_WEB_ADDR_MAP" env-delim:"," description:"Bot name to web address mapping"`

	LastFMKey string `long:"bot-lastfm-key" env:"HB_BOT_LASTFM_KEY" description:"LastFM API key"`
	SteamKey  string `long:"bot-steam-key" env:"HB_BOT_STEAM_KEY" description:"Steam API key"`

	Workers int `long:"bot-workers" env:"HB_BOT_WORKERS" description:"number of concurrent workers for handling"`

	PublicJoin bool `long:"bot-public-join" env:"HB_BOT_PUBLIC_JOIN" description:"enabled public join"`

	NoSend bool `long:"bot-no-send" env:"HB_BOT_NO_SEND" description:"log messages instead of sending them"`
}

var DefaultBot = Bot{
	DefaultCooldown: 5,
	WebAddr:         "http://localhost:5000",
	Workers:         runtime.GOMAXPROCS(0),
	PublicJoin:      true,
}

func (args *Bot) New(
	ctx context.Context,
	db *sql.DB,
	rdb *redis.DB,
	sender bot.Sender,
	notifier bot.Notifier,
	twitchAPI twitch.API,
) *bot.Bot {
	var lastFM lastfm.API
	if args.LastFMKey != "" {
		lastFM = lastfm.New(args.LastFMKey)
	} else {
		ctxlog.Warn(ctx, "no LastFM API key provided, functionality will be disabled")
	}

	var steamAPI steam.API
	if args.SteamKey != "" {
		steamAPI = steam.New(args.SteamKey)
	} else {
		ctxlog.Warn(ctx, "no Steam API key provided, functionality will be disabled")
	}

	if args.NoSend {
		sender = logSender{}
	}

	b := bot.New(&bot.Config{
		DB:               db,
		Redis:            rdb,
		Sender:           sender,
		Notifier:         notifier,
		LastFM:           lastFM,
		YouTube:          youtube.New(),
		XKCD:             xkcd.New(),
		ExtraLife:        extralife.New(),
		Twitch:           twitchAPI,
		Steam:            steamAPI,
		TinyURL:          tinyurl.New(),
		Admins:           args.Admins,
		WhitelistEnabled: args.WhitelistEnabled,
		Whitelist:        args.Whitelist,
		Cooldown:         args.DefaultCooldown,
		WebAddr:          args.WebAddr,
		WebAddrMap:       args.WebAddrMap,
		BulletMap:        args.BulletMap,
	})

	if err := b.Init(ctx); err != nil {
		ctxlog.Fatal(ctx, "error initializing bot", zap.Error(err))
	}

	return b
}

type logSender struct{}

func (logSender) SendMessage(ctx context.Context, origin, target, message string) error {
	ctxlog.Info(ctx, "not sending", zap.String("origin", origin), zap.String("target", target), zap.String("message", message))
	return nil
}
