package bot_test

import (
	"testing"

	"github.com/hortbot/hortbot/internal/bot"
	"github.com/hortbot/hortbot/internal/pkg/apis/twitch"
	"github.com/hortbot/hortbot/internal/pkg/dedupe"
	"github.com/hortbot/hortbot/internal/pkg/rdb"
	"github.com/hortbot/hortbot/internal/pkg/testutil/miniredistest"
	"gotest.tools/v3/assert"
)

func TestBotNewPanics(t *testing.T) {
	db, undb := freshDB(t)
	defer undb()

	_, rClient, rCleanup, err := miniredistest.New()
	assert.NilError(t, err)
	defer rCleanup()

	rDB, err := rdb.New(rClient)
	assert.NilError(t, err)

	config := &bot.Config{
		DB:       db,
		RDB:      rDB,
		Dedupe:   &struct{ dedupe.Deduplicator }{},
		Sender:   &struct{ bot.Sender }{},
		Notifier: &struct{ bot.Notifier }{},
		Twitch:   &struct{ twitch.API }{},
	}

	checkPanic := func() interface{} {
		var recovered interface{}

		func() {
			defer func() {
				recovered = recover()
			}()

			bot.New(config)
		}()

		return recovered
	}

	assert.Equal(t, checkPanic(), nil)

	config.DB = nil
	assert.Equal(t, checkPanic(), "db is nil")
	config.DB = db

	oldRedis := config.RDB
	config.RDB = nil
	assert.Equal(t, checkPanic(), "redis is nil")
	config.RDB = oldRedis

	oldDedupe := config.Dedupe
	config.Dedupe = nil
	assert.Equal(t, checkPanic(), "dedupe is nil")
	config.Dedupe = oldDedupe

	oldSender := config.Sender
	config.Sender = nil
	assert.Equal(t, checkPanic(), "sender is nil")
	config.Sender = oldSender

	oldNotifier := config.Notifier
	config.Notifier = nil
	assert.Equal(t, checkPanic(), "notifier is nil")
	config.Notifier = oldNotifier

	oldTwitch := config.Twitch
	config.Twitch = nil
	assert.Equal(t, checkPanic(), "twitch is nil")
	config.Twitch = oldTwitch

	rCleanup()
	assert.Equal(t, checkPanic(), nil)
}
