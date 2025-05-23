package bot_test

import (
	"testing"

	"github.com/hortbot/hortbot/internal/bot"
	"github.com/hortbot/hortbot/internal/db/redis"
	"github.com/hortbot/hortbot/internal/pkg/apiclient/hltb"
	"github.com/hortbot/hortbot/internal/pkg/apiclient/simple"
	"github.com/hortbot/hortbot/internal/pkg/apiclient/twitch"
	"github.com/hortbot/hortbot/internal/pkg/assertx"
)

func TestBotNewPanics(t *testing.T) {
	t.Parallel()
	db := pool.FreshDB(t)
	defer db.Close()

	config := &bot.Config{
		DB:                     db,
		Redis:                  &redis.DB{},
		EventsubUpdateNotifier: &struct{ bot.EventsubUpdateNotifier }{},
		Twitch:                 &struct{ twitch.API }{},
		Simple:                 &struct{ simple.API }{},
		HLTB:                   &struct{ hltb.API }{},
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

	oldEventsub := config.EventsubUpdateNotifier
	config.EventsubUpdateNotifier = nil
	assertx.Panic(t, checkPanic, "eventsub is nil")
	config.EventsubUpdateNotifier = oldEventsub

	oldTwitch := config.Twitch
	config.Twitch = nil
	assertx.Panic(t, checkPanic, "twitch is nil")
	config.Twitch = oldTwitch

	oldSimple := config.Simple
	config.Simple = nil
	assertx.Panic(t, checkPanic, "simple is nil")
	config.Simple = oldSimple

	oldHLTB := config.HLTB
	config.HLTB = nil
	assertx.Panic(t, checkPanic, "hltb is nil")
	config.HLTB = oldHLTB

	assertx.Panic(t, checkPanic, nil)
}

func TestBotNotInit(t *testing.T) {
	t.Parallel()
	assertx.Panic(t, func() {
		b := &bot.Bot{}
		b.Handle(t.Context(), privMSG("asdasd", "", 0, "", 0, ""))
	}, "bot is not initialized")
}
