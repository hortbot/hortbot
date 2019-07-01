package bot

import (
	"context"
	"database/sql"
	"log"
	"sync"
	"time"

	"github.com/efritz/glock"
	"github.com/go-redis/redis"
	"github.com/hortbot/hortbot/internal/db/models"
	"github.com/hortbot/hortbot/internal/pkg/dedupe"
	"github.com/hortbot/hortbot/internal/pkg/rdb"
	"github.com/hortbot/hortbot/internal/pkg/repeat"
	"github.com/volatiletech/sqlboiler/queries/qm"
)

const (
	DefaultPrefix = "!"
	DefaultBullet = "[HB]"
)

type Config struct {
	DB       *sql.DB
	Redis    redis.Cmdable
	Dedupe   dedupe.Deduplicator
	Sender   Sender
	Notifier Notifier
	Clock    glock.Clock
	Rand     Rand

	Prefix   string
	Bullet   string
	Cooldown int

	Admins []string

	WhitelistEnabled bool
	Whitelist        []string
}

type Bot struct {
	stopOnce sync.Once

	db   *sql.DB
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
	}

	deps := &sharedDeps{
		Dedupe:          config.Dedupe,
		Sender:          config.Sender,
		Notifier:        config.Notifier,
		DefaultPrefix:   config.Prefix,
		DefaultBullet:   config.Bullet,
		DefaultCooldown: config.Cooldown,
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
		deps.Clock = glock.NewRealClock()
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

	r, err := rdb.New(config.Redis)
	if err != nil {
		panic(err)
	}

	deps.RDB = r

	b := &Bot{
		db:   config.DB,
		deps: deps,
		rep:  repeat.New(nil, deps.Clock),
	}

	deps.UpdateRepeat = b.updateRepeatedCommand

	if isTesting {
		b.testingHelper = &testingHelper{}
	}

	return b
}

func (b *Bot) Init(ctx context.Context) error {
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

		log.Println(repeat.ID, delay, offset)

		b.updateRepeatedCommand(repeat.ID, true, delay, offset)
	}

	return nil
}

func (b *Bot) Stop() {
	b.stopOnce.Do(func() {
		b.rep.Stop()
	})
}
