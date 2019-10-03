// Package botargs processes bot arguments.
package botargs

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
	Admins []string `long:"admin" env:"HB_ADMINS" env-delim:"," description:"Bot admins"`

	WhitelistEnabled bool     `long:"whitelist-enabled" env:"HB_WHITELIST_ENABLED" description:"Enable the user whitelist"`
	Whitelist        []string `long:"whitelist" env:"HB_WHITELIST" env-delim:"," description:"User whitelist"`

	DefaultCooldown int `long:"default-cooldown" env:"HB_DEFAULT_COOLDOWN" description:"default command cooldown"`

	BotWebAddr    string            `long:"bot-web-addr" env:"HB_BOT_WEB_ADDR" description:"Default address for the bot website"`
	BotWebAddrMap map[string]string `long:"bot-web-addr-map" env:"HB_BOT_WEB_ADDR_MAP" description:"Bot name to web address mapping"`

	LastFMKey string `long:"lastfm-key" env:"HB_LASTFM_KEY" description:"LastFM API key"`
	SteamKey  string `long:"steam-key" env:"HB_STEAM_KEY" description:"Steam API key"`

	Workers int `long:"workers" env:"HB_WORKERS" description:"number of concurrent workers for handling"`
}

var DefaultBot = Bot{
	DefaultCooldown: 5,
	BotWebAddr:      "http://localhost:5000",
	Workers:         runtime.GOMAXPROCS(0),
}

func (args *Bot) NewBot(
	ctx context.Context,
	db *sql.DB,
	rdb *redis.DB,
	sender bot.Sender,
	notifier bot.Notifier,
	twitchAPI twitch.API,
) *bot.Bot {
	logger := ctxlog.FromContext(ctx)

	var lastFM lastfm.API
	if args.LastFMKey != "" {
		lastFM = lastfm.New(args.LastFMKey)
	} else {
		logger.Warn("no LastFM API key provided, functionality will be disabled")
	}

	var steamAPI steam.API
	if args.SteamKey != "" {
		steamAPI = steam.New(args.SteamKey)
	} else {
		logger.Warn("no Steam API key provided, functionality will be disabled")
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
		WebAddr:          args.BotWebAddr,
		WebAddrMap:       args.BotWebAddrMap,
	})

	if err := b.Init(ctx); err != nil {
		logger.Fatal("error initializing bot", zap.Error(err))
	}

	return b
}
