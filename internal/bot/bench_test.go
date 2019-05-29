package bot_test

import (
	"context"
	"strconv"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/hortbot/hortbot/internal/bot"
	"github.com/hortbot/hortbot/internal/ctxlog"
	"github.com/hortbot/hortbot/internal/db/models"
	"github.com/hortbot/hortbot/internal/dedupe"
	"github.com/hortbot/hortbot/internal/testutil"
	"github.com/hortbot/hortbot/internal/x/ircx"
	"github.com/volatiletech/sqlboiler/boil"
	"gotest.tools/assert"
)

func BenchmarkBot(b *testing.B) {
	db, undb := freshDB(b)
	defer undb()

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

	config := &bot.Config{
		DB:       db,
		Dedupe:   dedupe.NeverSeen,
		Sender:   nopSender{},
		Notifier: nopNotifier{},
	}

	bb := bot.New(config)

	m := ircx.PrivMsg("#"+name, "+pan working command")
	m.Tags = map[string]string{
		"id":      uuid.Must(uuid.NewV4()).String(),
		"room-id": strconv.FormatInt(channel.UserID, 10),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		bb.Handle(ctx, botName, m)
	}
}

type nopSender struct{}

func (nopSender) SendMessage(origin, target, message string) error { return nil }

type nopNotifier struct{}

func (nopNotifier) NotifyChannelUpdates(botName string) {}
