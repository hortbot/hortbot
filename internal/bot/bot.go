package bot

import (
	"context"
	"database/sql"

	"github.com/davecgh/go-spew/spew"
	"github.com/hortbot/hortbot/internal/ctxlog"
	"github.com/hortbot/hortbot/internal/db/models"
	"github.com/jakebailey/irc"
	"go.uber.org/zap"
)

type Config struct {
	DB *sql.DB
}

type Bot struct {
	config *Config
}

func NewBot(config *Config) *Bot {
	return &Bot{
		config: config,
	}
}

type MessageSender interface {
	SendMessage(target, message string) error
}

func (b *Bot) Handle(ctx context.Context, m *irc.Message) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	if m.Command != "PRIVMSG" {
		// TODO
		return
	}

	c := NewContext(m)

	spew.Dump(models.Channels(models.ChannelWhere.UserID.EQ(c.RoomID)).One(ctx, b.config.DB))

	ctx, logger := ctxlog.FromContextWith(ctx, zap.String("id", c.ID))

	defer func() {
		if r := recover(); r != nil {
			logger.Error("panic during handle", zap.Any("value", r), zap.Stack("stack"))
		}
	}()

	if err := b.handlePrivmsg(ctx, c); err != nil {
		logger.Error("handling message", zap.Error(err))
	}
}

func (b *Bot) handlePrivmsg(ctx context.Context, c *Context) error {
	logger := ctxlog.FromContext(ctx)
	logger.Info("handling message", zap.Any("context", c))
	return nil
}
