// Package botflags processes bot-related flags.
package botflags

import (
	"context"
	"database/sql"
	"net/http"
	"runtime"

	"github.com/hortbot/hortbot/internal/bot"
	"github.com/hortbot/hortbot/internal/db/redis"
	"github.com/hortbot/hortbot/internal/pkg/apiclient/extralife"
	"github.com/hortbot/hortbot/internal/pkg/apiclient/lastfm"
	"github.com/hortbot/hortbot/internal/pkg/apiclient/simple"
	"github.com/hortbot/hortbot/internal/pkg/apiclient/steam"
	"github.com/hortbot/hortbot/internal/pkg/apiclient/tinyurl"
	"github.com/hortbot/hortbot/internal/pkg/apiclient/twitch"
	"github.com/hortbot/hortbot/internal/pkg/apiclient/urban"
	"github.com/hortbot/hortbot/internal/pkg/apiclient/xkcd"
	"github.com/hortbot/hortbot/internal/pkg/apiclient/youtube"
	"github.com/hortbot/hortbot/internal/pkg/ctxlog"
	"go.uber.org/zap"
)

// Bot contains bot-specific flags.
type Bot struct {
	Admins      []string `long:"bot-admin" env:"HB_BOT_ADMINS" env-delim:"," description:"Bot admins"`
	SuperAdmins []string `long:"bot-super-admin" env:"HB_BOT_SUPER_ADMINS" env-delim:"," description:"Bot super admins"`

	WhitelistEnabled bool     `long:"bot-whitelist-enabled" env:"HB_BOT_WHITELIST_ENABLED" description:"Enable the user whitelist"`
	Whitelist        []string `long:"bot-whitelist" env:"HB_BOT_WHITELIST" env-delim:"," description:"User whitelist"`

	BulletMap       map[string]string `long:"bot-bullet-map" env:"HB_BOT_BULLET_MAP" env-delim:"," description:"Mapping from bot name to default bullet"`
	DefaultCooldown int               `long:"bot-default-cooldown" env:"HB_BOT_DEFAULT_COOLDOWN" description:"default command cooldown"`

	WebAddr    string            `long:"bot-web-addr" env:"HB_BOT_WEB_ADDR" description:"Default address for the bot website"`
	WebAddrMap map[string]string `long:"bot-web-addr-map" env:"HB_BOT_WEB_ADDR_MAP" env-delim:"," description:"Bot name to web address mapping"`

	LastFMKey  string `long:"bot-lastfm-key" env:"HB_BOT_LASTFM_KEY" description:"LastFM API key"`
	SteamKey   string `long:"bot-steam-key" env:"HB_BOT_STEAM_KEY" description:"Steam API key"`
	YouTubeKey string `long:"bot-youtube-key" env:"HB_BOT_YOUTUBE_KEY" description:"YouTube API key"`

	Workers int `long:"bot-workers" env:"HB_BOT_WORKERS" description:"number of concurrent workers for handling"`

	PublicJoin bool `long:"bot-public-join" env:"HB_BOT_PUBLIC_JOIN" description:"enabled public join"`

	NoSend bool `long:"bot-no-send" env:"HB_BOT_NO_SEND" description:"log messages instead of sending them"`
}

// Default contains the default flags. Make a copy of this, do not reuse.
var Default = Bot{
	DefaultCooldown: 5,
	WebAddr:         "http://localhost:5000",
	Workers:         runtime.GOMAXPROCS(0),
	PublicJoin:      true,
}

// New creates a new Bot from the set flags and dependencies.
func (args *Bot) New(
	ctx context.Context,
	db *sql.DB,
	rdb *redis.DB,
	sender bot.Sender,
	notifier bot.Notifier,
	twitchAPI twitch.API,
	httpClient *http.Client,
	untrustedClient *http.Client,
) *bot.Bot {
	var lastFM lastfm.API
	if args.LastFMKey != "" {
		lastFM = lastfm.New(args.LastFMKey, lastfm.HTTPClient(httpClient))
	} else {
		ctxlog.Warn(ctx, "no LastFM API key provided, functionality will be disabled")
	}

	var steamAPI steam.API
	if args.SteamKey != "" {
		steamAPI = steam.New(args.SteamKey, steam.HTTPClient(httpClient))
	} else {
		ctxlog.Warn(ctx, "no Steam API key provided, functionality will be disabled")
	}

	var youtubeAPI youtube.API
	if args.YouTubeKey != "" {
		youtubeAPI = youtube.New(args.YouTubeKey, youtube.HTTPClient(httpClient))
	} else {
		ctxlog.Warn(ctx, "no YouTube API key provided, functionality will be disabled")
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
		YouTube:          youtubeAPI,
		XKCD:             xkcd.New(xkcd.HTTPClient(httpClient)),
		ExtraLife:        extralife.New(extralife.HTTPClient(httpClient)),
		Twitch:           twitchAPI,
		Steam:            steamAPI,
		TinyURL:          tinyurl.New(tinyurl.HTTPClient(httpClient)),
		Urban:            urban.New(urban.HTTPClient(httpClient)),
		Simple:           simple.New(simple.HTTPClient(untrustedClient)),
		Admins:           args.Admins,
		SuperAdmins:      args.SuperAdmins,
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
