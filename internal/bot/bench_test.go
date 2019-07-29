package bot_test

import (
	"context"
	"fmt"
	"strconv"
	"sync/atomic"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/hortbot/hortbot/internal/bot"
	"github.com/hortbot/hortbot/internal/pkg/ctxlog"
	"github.com/hortbot/hortbot/internal/pkg/dedupe"
	"github.com/hortbot/hortbot/internal/pkg/ircx"
	"github.com/hortbot/hortbot/internal/pkg/testutil"
	"github.com/hortbot/hortbot/internal/pkg/testutil/miniredistest"
	"github.com/jakebailey/irc"
	"gotest.tools/assert"
)

func BenchmarkCustomCommand(b *testing.B) {
	const botName = "hortbot"

	_, rClient, rCleanup, err := miniredistest.New()
	assert.NilError(b, err)
	defer rCleanup()

	db, undb := freshDB(b)
	defer undb()

	ctx := ctxlog.WithLogger(context.Background(), testutil.Logger(b))

	userID, name := getNextUserID()

	config := &bot.Config{
		DB:       db,
		Redis:    rClient,
		Dedupe:   dedupe.NeverSeen,
		Sender:   nopSender{},
		Notifier: nopNotifier{},
	}

	bb := bot.New(config)

	m := ircx.PrivMsg("#"+botName, "!join")
	m.Tags = map[string]string{
		"id":      uuid.Must(uuid.NewV4()).String(),
		"room-id": "1",
		"user-id": strconv.FormatInt(userID, 10),
	}
	m.Prefix = irc.Prefix{
		Name: name,
		User: name,
		Host: name + "@tmi.twitch.tv",
	}

	bb.Handle(ctx, botName, m)

	m.Params = []string{"#" + name}
	m.Trailing = "!command add pan FOUND THE (_PARAMETER_CAPS_), HAVE YE?"
	m.Tags["room-id"] = m.Tags["user-id"]

	bb.Handle(ctx, botName, m)

	m.Trailing = "!pan working command"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		bb.Handle(ctx, botName, m)
	}
	b.StopTimer()
}

func BenchmarkNop(b *testing.B) {
	const botName = "hortbot"

	db, undb := freshDB(b)
	defer undb()

	ctx := ctxlog.WithLogger(context.Background(), testutil.Logger(b))

	_, rClient, rCleanup, err := miniredistest.New()
	assert.NilError(b, err)
	defer rCleanup()

	userID, name := getNextUserID()

	config := &bot.Config{
		DB:       db,
		Redis:    rClient,
		Dedupe:   dedupe.NeverSeen,
		Sender:   nopSender{},
		Notifier: nopNotifier{},
	}

	bb := bot.New(config)

	m := ircx.PrivMsg("#"+botName, "!join")
	m.Tags = map[string]string{
		"id":      uuid.Must(uuid.NewV4()).String(),
		"room-id": "1",
		"user-id": strconv.FormatInt(userID, 10),
	}
	m.Prefix = irc.Prefix{
		Name: name,
		User: name,
		Host: name + "@tmi.twitch.tv",
	}

	bb.Handle(ctx, botName, m)

	m.Params = []string{"#" + name}
	m.Trailing = "test"
	m.Tags["room-id"] = m.Tags["user-id"]

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		bb.Handle(ctx, botName, m)
	}
	b.StopTimer()
}

var nextUserID int64 = 1

func getNextUserID() (int64, string) {
	id := atomic.AddInt64(&nextUserID, 1)
	return id, fmt.Sprintf("user%d", id)
}

type nopSender struct{}

func (nopSender) SendMessage(origin, target, message string) error { return nil }

type nopNotifier struct{}

func (nopNotifier) NotifyChannelUpdates(botName string) {}
