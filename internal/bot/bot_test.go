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
	"github.com/hortbot/hortbot/internal/db/migrations"
	"github.com/hortbot/hortbot/internal/db/models"
	"github.com/hortbot/hortbot/internal/testutil"
	"github.com/hortbot/hortbot/internal/x/ircx"
	"github.com/ory/dockertest"
	"github.com/volatiletech/sqlboiler/boil"
	"gotest.tools/assert"

	_ "github.com/lib/pq"
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

	pool, err := dockertest.NewPool("")
	must(err)

	resource, err := pool.Run("zikaeroh/postgres-initialized", "latest", nil)
	must(err)

	defer func() {
		if err := pool.Purge(resource); err != nil {
			log.Println(err)
		}
	}()

	err = pool.Retry(func() error {
		var err error
		db, err = sql.Open("postgres", connStr(resource.GetHostPort("5432/tcp")))
		if err != nil {
			return err
		}

		return db.Ping()
	})
	must(err)

	defer db.Close()

	err = migrations.Up(db, nil)
	must(err)

	status = m.Run()
}

func TestBot(t *testing.T) {
	ctx := testutil.Logger(context.Background(), t)

	userID, name := getNextUserID()

	channel := &models.Channel{
		UserID: userID,
		Name:   name,
		Prefix: "+",
	}

	assert.NilError(t, channel.Insert(ctx, db, boil.Infer()))

	config := &Config{
		DB: db,
	}

	b := NewBot(config)

	m := ircx.PrivMsg("#foobar", "hi there")
	m.Tags = map[string]string{
		"id":      uuid.Must(uuid.NewV4()).String(),
		"room-id": strconv.FormatInt(channel.UserID, 10),
	}

	b.Handle(ctx, m)
}

func connStr(addr string) string {
	return fmt.Sprintf(`postgres://postgres:mysecretpassword@%s/postgres?sslmode=disable`, addr)
}
