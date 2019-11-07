package bot_test

import (
	"database/sql"
	"flag"
	"fmt"
	"os"
	"runtime"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/hortbot/hortbot/internal/bot"
	"github.com/hortbot/hortbot/internal/pkg/testutil/pgtest"
	"gotest.tools/v3/assert"
)

func init() {
	bot.Testing()
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}

type fdb struct {
	err     error
	db      *sql.DB
	cleanup func()
}

var (
	mainDB        *sql.DB
	mainConnStr   string
	conns         chan fdb
	concurrentDBs = false
)

func TestMain(m *testing.M) {
	var status int
	defer func() {
		os.Exit(status)
	}()

	flag.Parse()

	switch {
	case flag.Lookup("test.bench").Value.String() != "":
	case flag.Lookup("test.run").Value.String() != "":
	default:
		concurrentDBs = true
	}

	var cleanup func()
	var err error

	mainDB, mainConnStr, cleanup, err = pgtest.New()
	must(err)
	defer cleanup()

	// Create another database as a template because keeping the main connection
	// to the original database open prevents its use as a template.
	_, err = mainDB.Exec(`CREATE DATABASE temp_template WITH TEMPLATE postgres`)
	must(err)

	if concurrentDBs {
		procs := runtime.GOMAXPROCS(0)

		n := procs * 4
		if n > 16 {
			n = 16
		}

		conns = make(chan fdb, n)

		for i := 0; i < n; i++ {
			go dbMaker()
		}
	}

	status = m.Run()
}

var tempDBNum int64

func freshDB(t testing.TB) (*sql.DB, func()) {
	t.Helper()

	var f fdb
	if concurrentDBs {
		f = <-conns
	} else {
		f = makeDB()
	}

	assert.NilError(t, f.err)
	return f.db, f.cleanup
}

func dbMaker() {
	for {
		conns <- makeDB()
	}
}

func makeDB() fdb {
	dbName := fmt.Sprintf("temp%d", atomic.AddInt64(&tempDBNum, 1))

	_, err := mainDB.Exec(fmt.Sprintf(`CREATE DATABASE %s WITH TEMPLATE temp_template`, dbName))
	if err != nil {
		return fdb{err: err}
	}

	connStr := strings.Replace(mainConnStr, "postgres?", dbName+"?", 1)

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return fdb{err: err}
	}

	return fdb{
		db:      db,
		cleanup: func() { db.Close() },
	}
}
