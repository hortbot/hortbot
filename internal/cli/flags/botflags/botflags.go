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
	"github.com/hortbot/hortbot/internal/pkg/apiclient/hltb"
	"github.com/hortbot/hortbot/internal/pkg/apiclient/lastfm"
	"github.com/hortbot/hortbot/internal/pkg/apiclient/simple"
	"github.com/hortbot/hortbot/internal/pkg/apiclient/steam"
	"github.com/hortbot/hortbot/internal/pkg/apiclient/tinyurl"
	"github.com/hortbot/hortbot/internal/pkg/apiclient/twitch"
	"github.com/hortbot/hortbot/internal/pkg/apiclient/urban"
	"github.com/hortbot/hortbot/internal/pkg/apiclient/xkcd"
	"github.com/hortbot/hortbot/internal/pkg/apiclient/youtube"
	"github.com/zikaeroh/ctxlog"
	"go.uber.org/zap"
)

// Bot contains bot-specific flags.
type Bot struct {
	Admins      []string `long:"bot-admin" env:"HB_BOT_ADMINS" env-delim:"," description:"Bot admins"`
	SuperAdmins []string `long:"bot-super-admin" env:"HB_BOT_SUPER_ADMINS" env-delim:"," description:"Bot super admins"`

	WhitelistEnabled bool     `long:"bot-whitelist-enabled" env:"HB_BOT_WHITELIST_ENABLED" description:"Enable the user whitelist"`
	Whitelist        []string `long:"bot-whitelist" env:"HB_BOT_WHITELIST" env-delim:"," description:"User whitelist"`

	BulletMap       map[string]string `long:"bot-bullet-map" env:"HB_BOT_BULLET_MAP" env-delim:"," description:"Mapping from bot name to default bullet"`
	DefaultCooldown int               `long:"bot-default-cooldown" env:"HB_BOT_DEFAULT_COOLDOWN" description:"Default command cooldown"`

	WebAddr    string            `long:"bot-web-addr" env:"HB_BOT_WEB_ADDR" description:"Default address for the bot website"`
	WebAddrMap map[string]string `long:"bot-web-addr-map" env:"HB_BOT_WEB_ADDR_MAP" env-delim:"," description:"Bot name to web address mapping"`

	LastFMKey  string `long:"bot-lastfm-key" env:"HB_BOT_LASTFM_KEY" description:"LastFM API key"`
	SteamKey   string `long:"bot-steam-key" env:"HB_BOT_STEAM_KEY" description:"Steam API key"`
	YouTubeKey string `long:"bot-youtube-key" env:"HB_BOT_YOUTUBE_KEY" description:"YouTube API key"`

	Workers int `long:"bot-workers" env:"HB_BOT_WORKERS" description:"number of concurrent workers for handling"`

	PublicJoin         bool     `long:"bot-public-join" env:"HB_BOT_PUBLIC_JOIN" description:"Enable public join for all bots"`
	PublicJoinDisabled []string `long:"bot-public-join-disabled" env:"HB_BOT_PUBLIC_JOIN_DISABLED" env-delim:"," description:"Bots to disable public join on regardless of global public join setting"`

	GlobalIgnore []string `long:"bot-global-ignore" env:"HB_BOT_GLOBAL_IGNORE" env-delim:"," description:"List of users to ignore globally (e.g. known bots)"`
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
	eventsubUpdateNotifier bot.EventsubUpdateNotifier,
	twitchAPI twitch.API,
	httpClient *http.Client,
	untrustedClient *http.Client,
) *bot.Bot {
	var lastFM lastfm.API
	if args.LastFMKey != "" {
		lastFM = lastfm.New(args.LastFMKey, httpClient)
	} else {
		ctxlog.Warn(ctx, "no LastFM API key provided, functionality will be disabled")
	}

	var steamAPI steam.API
	if args.SteamKey != "" {
		steamAPI = steam.New(args.SteamKey, httpClient)
	} else {
		ctxlog.Warn(ctx, "no Steam API key provided, functionality will be disabled")
	}

	var youtubeAPI youtube.API
	if args.YouTubeKey != "" {
		youtubeAPI = youtube.New(args.YouTubeKey, httpClient)
	} else {
		ctxlog.Warn(ctx, "no YouTube API key provided, functionality will be disabled")
	}

	b := bot.New(&bot.Config{
		DB:                     db,
		Redis:                  rdb,
		EventsubUpdateNotifier: eventsubUpdateNotifier,
		LastFM:                 lastFM,
		YouTube:                youtubeAPI,
		XKCD:                   xkcd.New(httpClient),
		ExtraLife:              extralife.New(httpClient),
		Twitch:                 twitchAPI,
		Steam:                  steamAPI,
		TinyURL:                tinyurl.New(httpClient),
		Urban:                  urban.New(httpClient),
		Simple:                 simple.New(untrustedClient),
		HLTB:                   hltb.New(untrustedClient),
		Admins:                 args.Admins,
		SuperAdmins:            args.SuperAdmins,
		WhitelistEnabled:       args.WhitelistEnabled,
		Whitelist:              args.Whitelist,
		Cooldown:               args.DefaultCooldown,
		WebAddr:                args.WebAddr,
		WebAddrMap:             args.WebAddrMap,
		BulletMap:              args.BulletMap,
		PublicJoin:             args.PublicJoin,
		PublicJoinDisabled:     args.PublicJoinDisabled,
		GlobalIgnore:           args.GlobalIgnore,
		Cron: bot.CronConfig{
			ValidateTokens:          true,
			UpdateModeratedChannels: true,
		},
	})

	if err := b.Init(ctx); err != nil {
		ctxlog.Fatal(ctx, "error initializing bot", zap.Error(err))
	}

	return b
}
