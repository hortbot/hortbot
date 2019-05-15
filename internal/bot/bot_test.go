package bot_test

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"strconv"
	"sync/atomic"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/hortbot/hortbot/internal/bot"
	"github.com/hortbot/hortbot/internal/bot/botfakes"
	"github.com/hortbot/hortbot/internal/ctxlog"
	"github.com/hortbot/hortbot/internal/db/models"
	"github.com/hortbot/hortbot/internal/dedupe"
	"github.com/hortbot/hortbot/internal/testutil"
	"github.com/hortbot/hortbot/internal/testutil/pgtest"
	"github.com/hortbot/hortbot/internal/x/ircx"
	"github.com/volatiletech/sqlboiler/boil"
	"gotest.tools/assert"
)

const botName = "hortbot"

var nextUserID int64

func getNextUserID() (int64, string) {
	id := atomic.AddInt64(&nextUserID, 1)
	return id, fmt.Sprintf("user%d", id)
}

func must(err error) {
	if err != nil {
		log.Panic(err)
	}
}

var db *sql.DB

func TestMain(m *testing.M) {
	var status int
	defer func() {
		os.Exit(status)
	}()

	var closer func()
	var err error

	db, closer, err = pgtest.New()
	must(err)
	defer closer()

	status = m.Run()
}

func TestBot(t *testing.T) {
	ctx := ctxlog.WithLogger(context.Background(), testutil.Logger(t))

	userID, name := getNextUserID()

	channel := &models.Channel{
		UserID:  userID,
		Name:    name,
		Prefix:  "+",
		BotName: botName,
	}

	assert.NilError(t, channel.Insert(ctx, db, boil.Infer()))

	command := &models.SimpleCommand{
		ChannelID: channel.ID,
		Name:      "pan",
		Message:   "FOUND THE (_PARAMETER_CAPS_), HAVE YE?",
	}

	assert.NilError(t, command.Insert(ctx, db, boil.Infer()))

	sender := &botfakes.FakeMessageSender{}

	config := &bot.Config{
		DB:     db,
		Dedupe: dedupe.NeverSeen,
		Sender: sender,
		Name:   botName,
	}

	b := bot.NewBot(config)

	m := ircx.PrivMsg("#foobar", "+pan working command")
	m.Tags = map[string]string{
		"id":      uuid.Must(uuid.NewV4()).String(),
		"room-id": strconv.FormatInt(channel.UserID, 10),
	}

	b.Handle(ctx, m)

	assert.Equal(t, sender.SendMessageCallCount(), 1)

	target, message := sender.SendMessageArgsForCall(0)
	assert.Equal(t, target, "#foobar")
	assert.Equal(t, message, "[HB] FOUND THE WORKING COMMAND, HAVE YE?")
}

func BenchmarkBot(b *testing.B) {
	ctx := ctxlog.WithLogger(context.Background(), testutil.Logger(b))

	userID, name := getNextUserID()

	channel := &models.Channel{
		UserID:  userID,
		Name:    name,
		Prefix:  "+",
		BotName: botName,
	}

	assert.NilError(b, channel.Insert(ctx, db, boil.Infer()))

	command := &models.SimpleCommand{
		ChannelID: channel.ID,
		Name:      "pan",
		Message:   "FOUND THE (_PARAMETER_CAPS_), HAVE YE?",
	}

	assert.NilError(b, command.Insert(ctx, db, boil.Infer()))

	sender := &botfakes.FakeMessageSender{}

	config := &bot.Config{
		DB:     db,
		Dedupe: dedupe.NeverSeen,
		Sender: sender,
		Name:   botName,
	}

	bb := bot.NewBot(config)

	m := ircx.PrivMsg("#foobar", "+pan working command")
	m.Tags = map[string]string{
		"id":      uuid.Must(uuid.NewV4()).String(),
		"room-id": strconv.FormatInt(channel.UserID, 10),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		bb.Handle(ctx, m)
	}
}
