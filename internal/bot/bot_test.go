package bot_test

import (
	"context"
	"testing"

	"github.com/hortbot/hortbot/internal/bot"
	"github.com/hortbot/hortbot/internal/db/redis"
	"github.com/hortbot/hortbot/internal/pkg/apiclient/twitch"
	"github.com/hortbot/hortbot/internal/pkg/assertx"
	"github.com/jakebailey/irc"
)

func TestBotNewPanics(t *testing.T) {
	t.Parallel()
	db := pool.FreshDB(t)
	defer db.Close()

	config := &bot.Config{
		DB:       db,
		Redis:    &redis.DB{},
		Sender:   &struct{ bot.Sender }{},
		Notifier: &struct{ bot.Notifier }{},
		Twitch:   &struct{ twitch.API }{},
	}

	checkPanic := func() {
		bot.New(config)
	}

	assertx.Panic(t, checkPanic, nil)

	config.DB = nil
	assertx.Panic(t, checkPanic, "db is nil")
	config.DB = db

	oldRedis := config.Redis
	config.Redis = nil
	assertx.Panic(t, checkPanic, "redis is nil")
	config.Redis = oldRedis

	oldSender := config.Sender
	config.Sender = nil
	assertx.Panic(t, checkPanic, "sender is nil")
	config.Sender = oldSender

	oldNotifier := config.Notifier
	config.Notifier = nil
	assertx.Panic(t, checkPanic, "notifier is nil")
	config.Notifier = oldNotifier

	oldTwitch := config.Twitch
	config.Twitch = nil
	assertx.Panic(t, checkPanic, "twitch is nil")
	config.Twitch = oldTwitch

	assertx.Panic(t, checkPanic, nil)
}

func TestBotNotInit(t *testing.T) {
	t.Parallel()
	assertx.Panic(t, func() {
		b := &bot.Bot{}
		b.Handle(context.Background(), "asdasd", &irc.Message{})
	}, "bot is not initialized")
}
