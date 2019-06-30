package bot

import (
	"database/sql"

	"github.com/efritz/glock"
	"github.com/go-redis/redis"
	"github.com/hortbot/hortbot/internal/pkg/dedupe"
	"github.com/hortbot/hortbot/internal/pkg/rdb"
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
	db            *sql.DB
	deps          *sharedDeps
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
	}

	if isTesting {
		b.testingHelper = &testingHelper{}
	}

	return b
}
