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
	"github.com/hortbot/hortbot/internal/pkg/testutil/pgtest"
	"gotest.tools/assert"
)

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

	// Create another database as a template because keeping the main connection
	// to the original database open prevents its use as a template.
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
