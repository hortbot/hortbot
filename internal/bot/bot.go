package bot

import (
	"context"
	"database/sql"
	"sync"
	"time"

	"github.com/hortbot/hortbot/internal/db/dbtrace"
	"github.com/hortbot/hortbot/internal/db/models"
	"github.com/hortbot/hortbot/internal/db/redis"
	"github.com/hortbot/hortbot/internal/pkg/apis/extralife"
	"github.com/hortbot/hortbot/internal/pkg/apis/lastfm"
	"github.com/hortbot/hortbot/internal/pkg/apis/steam"
	"github.com/hortbot/hortbot/internal/pkg/apis/tinyurl"
	"github.com/hortbot/hortbot/internal/pkg/apis/twitch"
	"github.com/hortbot/hortbot/internal/pkg/apis/urban"
	"github.com/hortbot/hortbot/internal/pkg/apis/xkcd"
	"github.com/hortbot/hortbot/internal/pkg/apis/youtube"
	"github.com/hortbot/hortbot/internal/pkg/dedupe"
	"github.com/hortbot/hortbot/internal/pkg/recache"
	"github.com/hortbot/hortbot/internal/pkg/repeat"
	"github.com/leononame/clock"
	"github.com/opentracing/opentracing-go"
	"github.com/volatiletech/sqlboiler/queries/qm"
)

const (
	DefaultPrefix = "!"
	DefaultBullet = "[HB]"
)

type Config struct {
	DB       *sql.DB
	Redis    *redis.DB
	Dedupe   dedupe.Deduplicator
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

	Prefix   string
	Bullet   string
	Cooldown int

	Admins []string

	WhitelistEnabled bool
	Whitelist        []string
}

type Bot struct {
	initialized bool
	stopOnce    sync.Once

	db   dbtrace.ContextBeginExecutor
	deps *sharedDeps
	rep  *repeat.Repeater

	testingHelper *testingHelper
}

func New(config *Config) *Bot {
	// TODO: don't panic, return errors.
	switch {
	case config.DB == nil:
		panic("db is nil")
	case config.Redis == nil:
		panic("redis is nil")
	case config.Dedupe == nil:
		panic("dedupe is nil")
	case config.Sender == nil:
		panic("sender is nil")
	case config.Notifier == nil:
		panic("notifier is nil")
	case config.Twitch == nil:
		panic("twitch is nil")
	}

	deps := &sharedDeps{
		Redis:           config.Redis,
		Dedupe:          config.Dedupe,
		Sender:          config.Sender,
		Notifier:        config.Notifier,
		LastFM:          config.LastFM,
		DefaultPrefix:   config.Prefix,
		DefaultBullet:   config.Bullet,
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
	}

	if deps.DefaultBullet == "" {
		deps.DefaultBullet = DefaultBullet
	}

	if deps.DefaultPrefix == "" {
		deps.DefaultPrefix = DefaultPrefix
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
		db:   dbtrace.DB(config.DB),
		deps: deps,
	}

	deps.UpdateRepeat = b.updateRepeatedCommand
	deps.UpdateSchedule = b.updateScheduledCommand

	if isTesting {
		b.testingHelper = &testingHelper{}
	}

	return b
}

func (b *Bot) Init(ctx context.Context) error {
	span, ctx := opentracing.StartSpanFromContext(ctx, "Init")
	defer span.Finish()

	b.rep = repeat.New(ctx, b.deps.Clock)

	repeats, err := models.RepeatedCommands(
		qm.Select(models.RepeatedCommandColumns.ID, models.RepeatedCommandColumns.UpdatedAt, models.RepeatedCommandColumns.Delay),
		models.RepeatedCommandWhere.Enabled.EQ(true),
	).All(ctx, b.db)
	if err != nil {
		return err
	}

	for _, repeat := range repeats {
		delay := time.Duration(repeat.Delay) * time.Second
		delayNano := delay.Nanoseconds()

		sinceUpdateNano := b.deps.Clock.Since(repeat.UpdatedAt).Nanoseconds()

		offsetNano := delayNano - sinceUpdateNano%delayNano
		offset := time.Duration(offsetNano) * time.Nanosecond

		b.updateRepeatedCommand(repeat.ID, true, delay, offset)
	}

	scheduleds, err := models.ScheduledCommands(
		qm.Select(models.ScheduledCommandColumns.ID, models.ScheduledCommandColumns.UpdatedAt, models.ScheduledCommandColumns.CronExpression),
		models.ScheduledCommandWhere.Enabled.EQ(true),
	).All(ctx, b.db)
	if err != nil {
		return err
	}

	for _, scheduled := range scheduleds {
		expr, err := repeat.ParseCron(scheduled.CronExpression)
		if err != nil {
			panic(err)
		}
		b.updateScheduledCommand(scheduled.ID, true, expr)
	}

	b.initialized = true

	return nil
}

func (b *Bot) Stop() {
	b.stopOnce.Do(func() {
		b.rep.Stop()
	})
}
