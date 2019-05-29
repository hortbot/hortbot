package bot_test

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/hortbot/hortbot/internal/bot"
	"github.com/hortbot/hortbot/internal/testutil/pgtest"
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

var mainDB *sql.DB
var mainConnStr string

func TestMain(m *testing.M) {
	var status int
	defer func() {
		os.Exit(status)
	}()

	var cleanup func()
	var err error

	mainDB, mainConnStr, cleanup, err = pgtest.New()
	must(err)
	defer cleanup()

	_, err = mainDB.Exec(`CREATE DATABASE temp_template WITH TEMPLATE postgres`)
	must(err)

	status = m.Run()
}

var tempDBNum int64

func freshDB(t testing.TB) (*sql.DB, func()) {
	t.Helper()

	dbName := fmt.Sprintf("temp%d", atomic.AddInt64(&tempDBNum, 1))

	_, err := mainDB.Exec(fmt.Sprintf(`CREATE DATABASE %s WITH TEMPLATE temp_template`, dbName))
	assert.NilError(t, err)

	connStr := strings.Replace(mainConnStr, "postgres?", dbName+"?", 1)

	db, err := sql.Open("postgres", connStr)
	assert.NilError(t, err)

	return db, func() {
		t.Helper()
		assert.NilError(t, db.Close())
	}
}

func init() {
	bot.Testing()
}
