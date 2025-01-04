// Package pgpool implements a pool of databases for testing.
package pgpool

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/hortbot/hortbot/internal/db/driver"
	"github.com/hortbot/hortbot/internal/db/migrations"
	"github.com/hortbot/hortbot/internal/pkg/docker/dpostgres"
	"gotest.tools/v3/assert"
)

// Pool provides access to a pool of fresh databases for testing.
// Its zero value is ready to use. If the pool is never used, then no
// database will be created.
type Pool struct {
	once sync.Once
	err  error

	pdb *dpostgres.DB

	db      *sql.DB
	connStr string

	num atomic.Int64
}

func (p *Pool) init(t testing.TB) {
	t.Helper()

	p.once.Do(func() {
		p.err = func() error {
			var err error
			p.pdb, err = dpostgres.New()
			if err != nil {
				return fmt.Errorf("creating database: %w", err)
			}

			p.connStr = p.pdb.ConnStr()

			if err := migrations.Up(p.connStr, t.Logf); err != nil {
				return fmt.Errorf("migrating database: %w", err)
			}

			p.db, err = p.pdb.Open()
			if err != nil {
				return fmt.Errorf("opening database: %w", err)
			}

			// Create another database as a template because keeping the main connection
			// to the original database open prevents its use as a template.
			_, err = p.db.Exec(`CREATE DATABASE temp_template WITH TEMPLATE postgres`)
			if err != nil {
				return fmt.Errorf("creating template database: %w", err)
			}
			return nil
		}()
	})

	assert.NilError(t, p.err, "initializing database")
}

// Cleanup cleans up the pool. It's safe to run, even if the pool was never used.
func (p *Pool) Cleanup() {
	p.once.Do(func() {
		p.err = errors.New("pgpool: cleaned up")
	})

	if p.pdb != nil {
		p.pdb.Cleanup()
	}
}

// FreshDB creates a new database, migrated up.
func (p *Pool) FreshDB(t testing.TB) *sql.DB {
	t.Helper()
	p.init(t)

	dbName := fmt.Sprintf("temp%d", p.num.Add(1))

	_, err := p.db.Exec(fmt.Sprintf(`CREATE DATABASE %s WITH TEMPLATE temp_template`, dbName))
	assert.NilError(t, err, "creating temp database")

	connStr := strings.Replace(p.connStr, "postgres?", dbName+"?", 1)

	db, err := sql.Open(driver.Name, connStr)
	assert.NilError(t, err, "opening temp database")

	return db
}
