// Package bot implements the core HortBot IRC message handling logic.
package bot

import (
	"context"
	"database/sql"
	"strings"
	"sync"
	"time"

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
	"github.com/hortbot/hortbot/internal/pkg/errgroupx"
	"github.com/hortbot/hortbot/internal/pkg/recache"
	"github.com/hortbot/hortbot/internal/pkg/repeat"
	"github.com/leononame/clock"
	"go.opencensus.io/trace"
)

const (
	// DefaultBullet is the default bullet used when the channel's bullet is unset.
	DefaultBullet = "[HB]"
)

// Config configures the bot.
type Config struct {
	DB       *sql.DB
	Redis    *redis.DB
	Sender   Sender
	Notifier Notifier
	Clock    clock.Clock
	Rand     Rand

	LastFM    lastfm.API
	YouTube   youtube.API
	XKCD      xkcd.API
	ExtraLife extralife.API
	Twitch    twitch.API
	Steam     steam.API
	TinyURL   tinyurl.API
	Urban     urban.API
	Simple    simple.API
	HLTB      hltb.API

	BulletMap map[string]string
	Cooldown  int

	Admins      []string
	SuperAdmins []string

	WhitelistEnabled bool
	Whitelist        []string

	WebAddr    string
	WebAddrMap map[string]string

	NoDedupe bool

	PublicJoin         bool
	PublicJoinDisabled []string

	BetaFeatures []string

	GlobalIgnore []string

	ValidateTokens bool
}

// Bot is an IRC bot. It should only be used once.
type Bot struct {
	initialized bool
	stopOnce    sync.Once
	g           *errgroupx.Group

	db   *sql.DB
	deps *sharedDeps
	rep  *repeat.Repeater

	validateTokensTicker clock.Ticker
	validateTokensManual chan struct{}

	testingHelper *testingHelper

	noDedupe bool
}

// New creates a new Bot with the given config.
func New(config *Config) *Bot {
	switch {
	case config.DB == nil:
		panic("db is nil")
	case config.Redis == nil:
		panic("redis is nil")
	case config.Sender == nil:
		panic("sender is nil")
	case config.Notifier == nil:
		panic("notifier is nil")
	case config.Twitch == nil:
		panic("twitch is nil")
	case config.Simple == nil:
		panic("simple is nil")
	case config.HLTB == nil:
		panic("hltb is nil")
	}

	deps := &sharedDeps{
		Redis:              config.Redis,
		Sender:             config.Sender,
		Notifier:           config.Notifier,
		LastFM:             config.LastFM,
		BulletMap:          config.BulletMap,
		DefaultCooldown:    config.Cooldown,
		YouTube:            config.YouTube,
		XKCD:               config.XKCD,
		ExtraLife:          config.ExtraLife,
		Twitch:             config.Twitch,
		Steam:              config.Steam,
		TinyURL:            config.TinyURL,
		Urban:              config.Urban,
		Simple:             config.Simple,
		HLTB:               config.HLTB,
		ReCache:            recache.New(),
		Admins:             make(map[string]bool),
		SuperAdmins:        make(map[string]bool),
		WebAddr:            config.WebAddr,
		WebAddrMap:         config.WebAddrMap,
		PublicJoin:         config.PublicJoin,
		PublicJoinDisabled: config.PublicJoinDisabled,
		BetaFeatures:       config.BetaFeatures,
		GlobalIgnore:       make(map[string]bool),
	}

	if config.Clock != nil {
		deps.Clock = config.Clock
	} else {
		deps.Clock = clock.New()
	}

	for _, name := range config.Admins {
		deps.Admins[name] = true
	}

	for _, name := range config.SuperAdmins {
		deps.Admins[name] = true
		deps.SuperAdmins[name] = true
	}

	if config.WhitelistEnabled {
		deps.Whitelist = make(map[string]bool)
		for _, name := range config.Whitelist {
			deps.Whitelist[name] = true
		}
	}

	for _, name := range config.GlobalIgnore {
		name = strings.ToLower(name)
		deps.GlobalIgnore[name] = true
	}

	if config.Rand != nil {
		deps.Rand = config.Rand
	} else {
		deps.Rand = pooledRand{}
	}

	b := &Bot{
		db:                   config.DB,
		deps:                 deps,
		noDedupe:             config.NoDedupe,
		rep:                  repeat.New(deps.Clock),
		validateTokensManual: make(chan struct{}, 1),
	}

	if config.ValidateTokens {
		b.validateTokensTicker = deps.Clock.NewTicker(time.Hour)
	} else {
		b.validateTokensTicker = noopTicker{}
	}

	deps.AddRepeat = b.addRepeat
	deps.RemoveRepeat = b.removeRepeat
	deps.AddScheduled = b.addScheduled
	deps.RemoveScheduled = b.removeScheduled
	deps.ReloadRepeats = b.loadRepeats
	deps.CountRepeats = b.rep.Count
	deps.TriggerValidateTokens = b.triggerValidateTokensNow

	if isTesting {
		b.testingHelper = &testingHelper{}
	}

	return b
}

var _ clock.Ticker = noopTicker{}

type noopTicker struct{}

func (noopTicker) Chan() <-chan time.Time { return nil }

func (noopTicker) Stop() {}

// Init initializes the bot, starting any underlying tasks. It should only be
// called once.
func (b *Bot) Init(ctx context.Context) error {
	ctx, span := trace.StartSpan(ctx, "Init")
	defer span.End()

	b.g = errgroupx.FromContext(ctx)
	b.g.Go(b.rep.Run)
	b.g.Go(b.runValidateTokens)

	if err := b.loadRepeats(ctx); err != nil {
		return err
	}

	b.initialized = true
	return nil
}

// Stop instructs the bot to stop.
func (b *Bot) Stop() {
	b.stopOnce.Do(func() {
		if g := b.g; g != nil {
			g.Stop()
		}
		b.validateTokensTicker.Stop()
	})
}
