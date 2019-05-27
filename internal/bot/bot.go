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
	DB     *sql.DB
	Dedupe dedupe.Deduplicator
	Sender MessageSender

	Prefix string
	Bullet string
}

type Bot struct {
	db     *sql.DB
	dedupe dedupe.Deduplicator
	sender MessageSender

	prefix string
	bullet string

	testingHelper testingHelper
}

func New(config *Config) *Bot {
	b := &Bot{
		db:     config.DB,
		dedupe: config.Dedupe,
		sender: config.Sender,
		prefix: config.Prefix,
		bullet: config.Bullet,
	}

	// TODO: verify other dependencies

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

//go:generate gobin -m -run github.com/maxbrunsfeld/counterfeiter/v6 . MessageSender

type MessageSender interface {
	SendMessage(origin, target, message string) error
}

type MessageSenderFunc func(origin, target, message string) error

func (f MessageSenderFunc) SendMessage(origin, target, message string) error {
	return f(origin, target, message)
}
