package bot

import (
	"database/sql"
	"strings"

	"github.com/hortbot/hortbot/internal/dedupe"
)

const DefaultBullet = "[HB]"

type Config struct {
	DB     *sql.DB
	Dedupe dedupe.Deduplicator
	Sender MessageSender

	Name   string
	Bullet string
}

type Bot struct {
	db     *sql.DB
	dedupe dedupe.Deduplicator
	sender MessageSender

	name   string
	bullet string
}

func NewBot(config *Config) *Bot {
	b := &Bot{
		db:     config.DB,
		dedupe: config.Dedupe,
		sender: config.Sender,
		name:   strings.ToLower(config.Name),
		bullet: config.Bullet,
	}

	// TODO: verify other dependencies

	if b.name == "" {
		panic("empty name") // TODO: error
	}

	if b.bullet == "" {
		b.bullet = DefaultBullet
	}

	return b
}

//go:generate gobin -m -run github.com/maxbrunsfeld/counterfeiter/v6 . MessageSender

type MessageSender interface {
	SendMessage(target, message string) error
}
