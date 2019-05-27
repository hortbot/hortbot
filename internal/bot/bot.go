package bot

import (
	"database/sql"

	"github.com/hortbot/hortbot/internal/dedupe"
)

const (
	DefaultPrefix = "!"
	DefaultBullet = "[HB]"
)

type Config struct {
	DB       *sql.DB
	Dedupe   dedupe.Deduplicator
	Sender   Sender
	Notifier Notifier

	Prefix string
	Bullet string
}

type Bot struct {
	db       *sql.DB
	dedupe   dedupe.Deduplicator
	sender   Sender
	notifier Notifier

	prefix string
	bullet string

	testingHelper testingHelper
}

func New(config *Config) *Bot {
	switch {
	case config.DB == nil:
		panic("db is nil")
	case config.Dedupe == nil:
		panic("dedupe is nil")
	case config.Sender == nil:
		panic("sender is nil")
	case config.Notifier == nil:
		panic("notifier is nil")
	}

	b := &Bot{
		db:       config.DB,
		dedupe:   config.Dedupe,
		sender:   config.Sender,
		notifier: config.Notifier,
		prefix:   config.Prefix,
		bullet:   config.Bullet,
	}

	if b.bullet == "" {
		b.bullet = DefaultBullet
	}

	if b.prefix == "" {
		b.prefix = DefaultPrefix
	}

	if isTesting {
		b.testingHelper = testingHelper{}
	}

	return b
}
