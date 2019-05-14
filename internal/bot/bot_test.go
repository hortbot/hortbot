package bot

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
	"github.com/hortbot/hortbot/internal/ctxlog"
	"github.com/hortbot/hortbot/internal/db/models"
	"github.com/hortbot/hortbot/internal/testutil"
	"github.com/hortbot/hortbot/internal/testutil/pgtest"
	"github.com/hortbot/hortbot/internal/x/ircx"
	"github.com/volatiletech/sqlboiler/boil"
	"gotest.tools/assert"
)

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
		UserID: userID,
		Name:   name,
		Prefix: "+",
	}

	assert.NilError(t, channel.Insert(ctx, db, boil.Infer()))

	command := &models.SimpleCommand{
		ChannelID: channel.ID,
		Name:      "pan",
		Message:   "FOUND THE (_PARAMETER_CAPS_), HAVE YE?",
	}

	assert.NilError(t, command.Insert(ctx, db, boil.Infer()))

	config := &Config{
		DB: db,
	}

	b := NewBot(config)

	m := ircx.PrivMsg("#foobar", "+pan working command")
	m.Tags = map[string]string{
		"id":      uuid.Must(uuid.NewV4()).String(),
		"room-id": strconv.FormatInt(channel.UserID, 10),
	}

	b.Handle(ctx, m)
}

func connStr(addr string) string {
	return fmt.Sprintf(`postgres://postgres:mysecretpassword@%s/postgres?sslmode=disable`, addr)
}

func BenchmarkBot(b *testing.B) {
	ctx := context.Background()

	userID, name := getNextUserID()

	channel := &models.Channel{
		UserID: userID,
		Name:   name,
		Prefix: "+",
	}

	assert.NilError(b, channel.Insert(ctx, db, boil.Infer()))

	config := &Config{
		DB: db,
	}

	bot := NewBot(config)

	m := ircx.PrivMsg("#foobar", "hi there")
	m.Tags = map[string]string{
		"id":      uuid.Must(uuid.NewV4()).String(),
		"room-id": strconv.FormatInt(channel.UserID, 10),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		bot.Handle(ctx, m)
	}
}
