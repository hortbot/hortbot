package bot

import (
	"context"
	"database/sql"
	"sync"

	"github.com/hortbot/hortbot/internal/db/redis"
	"github.com/hortbot/hortbot/internal/pkg/apis/extralife"
	"github.com/hortbot/hortbot/internal/pkg/apis/lastfm"
	"github.com/hortbot/hortbot/internal/pkg/apis/steam"
	"github.com/hortbot/hortbot/internal/pkg/apis/tinyurl"
	"github.com/hortbot/hortbot/internal/pkg/apis/twitch"
	"github.com/hortbot/hortbot/internal/pkg/apis/urban"
	"github.com/hortbot/hortbot/internal/pkg/apis/xkcd"
	"github.com/hortbot/hortbot/internal/pkg/apis/youtube"
	"github.com/hortbot/hortbot/internal/pkg/recache"
	"github.com/hortbot/hortbot/internal/pkg/repeat"
	"github.com/leononame/clock"
	"go.opencensus.io/trace"
)

const (
	DefaultBullet = "[HB]"
)

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

	BulletMap map[string]string
	Cooldown  int

	Admins []string

	WhitelistEnabled bool
	Whitelist        []string

	WebAddr    string
	WebAddrMap map[string]string

	NoDedupe bool

	PublicJoin bool
}

type Bot struct {
	initialized bool
	stopOnce    sync.Once

	db   *sql.DB
	deps *sharedDeps
	rep  *repeat.Repeater

	testingHelper *testingHelper

	noDedupe bool
}

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
	}

	deps := &sharedDeps{
		Redis:           config.Redis,
		Sender:          config.Sender,
		Notifier:        config.Notifier,
		LastFM:          config.LastFM,
		BulletMap:       config.BulletMap,
		DefaultCooldown: config.Cooldown,
		YouTube:         config.YouTube,
		XKCD:            config.XKCD,
		ExtraLife:       config.ExtraLife,
		Twitch:          config.Twitch,
		Steam:           config.Steam,
		TinyURL:         config.TinyURL,
		Urban:           config.Urban,
		ReCache:         recache.New(),
		Admins:          make(map[string]bool),
		WebAddr:         config.WebAddr,
		WebAddrMap:      config.WebAddrMap,
		PublicJoin:      config.PublicJoin,
	}

	if config.Clock != nil {
		deps.Clock = config.Clock
	} else {
		deps.Clock = clock.New()
	}

	for _, name := range config.Admins {
		deps.Admins[name] = true
	}

	if config.WhitelistEnabled {
		deps.Whitelist = make(map[string]bool)
		for _, name := range config.Whitelist {
			deps.Whitelist[name] = true
		}
	}

	if config.Rand != nil {
		deps.Rand = config.Rand
	} else {
		deps.Rand = globalRand{}
	}

	b := &Bot{
		db:       config.DB,
		deps:     deps,
		noDedupe: config.NoDedupe,
	}

	deps.UpdateRepeat = b.updateRepeatedCommand
	deps.UpdateSchedule = b.updateScheduledCommand
	deps.ReloadRepeats = func(ctx context.Context) error {
		return b.loadRepeats(ctx, true)
	}
	deps.CountRepeats = func() (int, int) {
		return b.rep.Count()
	}

	if isTesting {
		b.testingHelper = &testingHelper{}
	}

	return b
}

func (b *Bot) Init(ctx context.Context) error {
	ctx, span := trace.StartSpan(ctx, "Init")
	defer span.End()

	b.rep = repeat.New(ctx, b.deps.Clock)
	if err := b.loadRepeats(ctx, false); err != nil {
		return err
	}

	b.initialized = true
	return nil
}

func (b *Bot) Stop() {
	b.stopOnce.Do(func() {
		b.rep.Stop()
	})
}
