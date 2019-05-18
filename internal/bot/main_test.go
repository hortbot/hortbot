package bot_test

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"sync/atomic"
	"testing"

	"github.com/hortbot/hortbot/internal/testutil/pgtest"
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
var pgConnStr string

func TestMain(m *testing.M) {
	var status int
	defer func() {
		os.Exit(status)
	}()

	var cleanup func()
	var err error

	db, pgConnStr, cleanup, err = pgtest.New()
	must(err)
	defer cleanup()

	status = m.Run()
}
