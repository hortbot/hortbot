package bot_test

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"sync/atomic"
	"testing"

	"github.com/hortbot/hortbot/internal/bot"
	"github.com/hortbot/hortbot/internal/db/migrations"
	"github.com/hortbot/hortbot/internal/testutil/pgtest"
	"gotest.tools/assert"
)

const pgConns = 4

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

type sqlDB struct {
	db      *sql.DB
	connStr string
}

var dbs chan sqlDB

func freshDB(t testing.TB) (*sql.DB, func()) {
	db := <-dbs

	assert.NilError(t, migrations.Reset(db.connStr, nil))

	return db.db, func() {
		dbs <- db
	}
}

func anyDB() (*sql.DB, func()) {
	db := <-dbs
	return db.db, func() {
		dbs <- db
	}
}

func TestMain(m *testing.M) {
	var status int
	defer func() {
		os.Exit(status)
	}()

	cleanups := make([]func(), pgConns)
	defer func() {
		for _, f := range cleanups {
			if f != nil {
				defer f()
			}
		}
	}()

	dbs = make(chan sqlDB, pgConns)

	for i := 0; i < pgConns; i++ {
		i := i

		go func() {
			db, connStr, cleanup, err := pgtest.New()
			must(err)

			cleanups[i] = cleanup
			dbs <- sqlDB{
				db:      db,
				connStr: connStr,
			}
		}()
	}

	status = m.Run()
}

func init() {
	bot.Testing()
}
