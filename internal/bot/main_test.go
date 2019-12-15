package bot_test

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/hortbot/hortbot/internal/bot"
	"github.com/hortbot/hortbot/internal/pkg/docker/dpostgres"
	"gotest.tools/v3/assert"

	_ "github.com/jackc/pgx/v4/stdlib" // For postgres.
)

func init() {
	bot.Testing()
}

func TestMain(m *testing.M) {
	status := 1
	defer func() {
		if r := recover(); r != nil {
			panic(r)
		}
		os.Exit(status)
	}()

	defer func() {
		if cleanupDB != nil {
			cleanupDB()
		}
	}()

	status = m.Run()
}

var (
	mainDB      *sql.DB
	mainConnStr string
	cleanupDB   func()
	initDBOnce  sync.Once
	tempDBNum   int64
	tempDBSema  = make(chan bool, maxConcurrentCreates())
)

func maxConcurrentCreates() int64 {
	n := runtime.GOMAXPROCS(0) * 4
	if n > 16 {
		return 16
	}
	return int64(n)
}

func initDB() {
	var err error
	mainDB, mainConnStr, cleanupDB, err = dpostgres.New()
	if err != nil {
		log.Fatal(err)
	}

	// Create another database as a template because keeping the main connection
	// to the original database open prevents its use as a template.
	_, err = mainDB.Exec(`CREATE DATABASE temp_template WITH TEMPLATE postgres`)
	if err != nil {
		log.Fatal(err)
	}
}

func freshDB(t testing.TB) *sql.DB {
	t.Helper()
	initDBOnce.Do(initDB)

	tempDBSema <- true
	defer func() {
		<-tempDBSema
	}()

	dbName := fmt.Sprintf("temp%d", atomic.AddInt64(&tempDBNum, 1))

	_, err := mainDB.Exec(fmt.Sprintf(`CREATE DATABASE %s WITH TEMPLATE temp_template`, dbName))
	assert.NilError(t, err)

	connStr := strings.Replace(mainConnStr, "postgres?", dbName+"?", 1)

	db, err := sql.Open("pgx", connStr)
	assert.NilError(t, err)

	return db
}
