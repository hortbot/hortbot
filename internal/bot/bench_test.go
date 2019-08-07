package bot_test

import (
	"context"
	"fmt"
	"strconv"
	"sync/atomic"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/hortbot/hortbot/internal/bot"
	"github.com/hortbot/hortbot/internal/pkg/apis/twitch/twitchfakes"
	"github.com/hortbot/hortbot/internal/pkg/ctxlog"
	"github.com/hortbot/hortbot/internal/pkg/dedupe"
	"github.com/hortbot/hortbot/internal/pkg/rdb"
	"github.com/hortbot/hortbot/internal/pkg/testutil"
	"github.com/hortbot/hortbot/internal/pkg/testutil/miniredistest"
	"github.com/jakebailey/irc"
	"gotest.tools/assert"
)

func BenchmarkNop(b *testing.B) {
	const botName = "hortbot"

	db, undb := freshDB(b)
	defer undb()

	ctx := ctxlog.WithLogger(context.Background(), testutil.Logger(b))

	_, rClient, rCleanup, err := miniredistest.New()
	assert.NilError(b, err)
	defer rCleanup()

	rDB, err := rdb.New(rClient)
	assert.NilError(b, err)

	userID, name := getNextUserID()

	config := &bot.Config{
		DB:       db,
		RDB:      rDB,
		Dedupe:   dedupe.NeverSeen,
		Sender:   nopSender{},
		Notifier: nopNotifier{},
		Twitch:   &twitchfakes.FakeAPI{},
	}

	bb := bot.New(config)

	bb.Handle(ctx, botName, privMSG(botName, 1, name, userID, "!join"))

	m := privMSG(name, userID, name, userID, "test")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		bb.Handle(ctx, botName, m)
	}
	b.StopTimer()
}

func BenchmarkCustomCommand(b *testing.B) {
	const botName = "hortbot"

	_, rClient, rCleanup, err := miniredistest.New()
	assert.NilError(b, err)
	defer rCleanup()

	rDB, err := rdb.New(rClient)
	assert.NilError(b, err)

	db, undb := freshDB(b)
	defer undb()

	ctx := ctxlog.WithLogger(context.Background(), testutil.Logger(b))

	userID, name := getNextUserID()

	config := &bot.Config{
		DB:       db,
		RDB:      rDB,
		Dedupe:   dedupe.NeverSeen,
		Sender:   nopSender{},
		Notifier: nopNotifier{},
		Twitch:   &twitchfakes.FakeAPI{},
	}

	bb := bot.New(config)

	bb.Handle(ctx, botName, privMSG(botName, 1, name, userID, "!join"))
	bb.Handle(ctx, botName, privMSG(name, userID, name, userID, "!command add pan FOUND THE (_PARAMETER_CAPS_), HAVE YE?"))

	m := privMSG(name, userID, name, userID, "!pan working command")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		bb.Handle(ctx, botName, m)
	}
	b.StopTimer()
}

func BenchmarkMixed(b *testing.B) {
	const botName = "hortbot"

	_, rClient, rCleanup, err := miniredistest.New()
	assert.NilError(b, err)
	defer rCleanup()

	rDB, err := rdb.New(rClient)
	assert.NilError(b, err)

	db, undb := freshDB(b)
	defer undb()

	ctx := ctxlog.WithLogger(context.Background(), testutil.Logger(b))

	userID, name := getNextUserID()

	config := &bot.Config{
		DB:       db,
		RDB:      rDB,
		Dedupe:   dedupe.NeverSeen,
		Sender:   nopSender{},
		Notifier: nopNotifier{},
		Twitch:   &twitchfakes.FakeAPI{},
	}

	bb := bot.New(config)

	bb.Handle(ctx, botName, privMSG(botName, 1, name, userID, "!join"))
	bb.Handle(ctx, botName, privMSG(name, userID, name, userID, "!command add pan FOUND THE (_PARAMETER_CAPS_), HAVE YE?"))
	bb.Handle(ctx, botName, privMSG(name, userID, name, userID, "!autoreply add *who_is_zik* Nobody important."))
	bb.Handle(ctx, botName, privMSG(name, userID, name, userID, `!autoreply add REGEX:(^|\b)wowee($|\b) Wowee`))

	var ms []*irc.Message

	for i := 0; i < 95; i++ {
		ms = append(ms, privMSG(name, userID, "someone", 9999999, "nothing interesting"))
	}

	ms = append(ms,
		privMSG(name, userID, name, userID, "!pan working command"),
		privMSG(name, userID, name, userID, "who is zik"),
		privMSG(name, userID, name, userID, "!who knows"),
		privMSG(name, userID, name, userID, "!admin"),
		privMSG(name, userID, name, userID, "!set prefix"),
	)

	l := len(ms)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m := ms[i%l]
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

func privMSG(ch string, roomID int64, user string, userID int64, msg string) *irc.Message {
	return &irc.Message{
		Tags: map[string]string{
			"id":      uuid.Must(uuid.NewV4()).String(),
			"room-id": strconv.FormatInt(roomID, 10),
			"user-id": strconv.FormatInt(userID, 10),
		},
		Prefix: irc.Prefix{
			Name: user,
			User: user,
			Host: user + "@tmi.twitch.tv",
		},
		Command:  "PRIVMSG",
		Params:   []string{"#" + ch},
		Trailing: msg,
	}
}
