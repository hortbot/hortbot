// Package pgpool implements a pool of databases for testing.
package pgpool

import (
	"database/sql"
	"errors"
	"fmt"
	"sync"
	"testing"

	"github.com/hortbot/hortbot/internal/db/migrations"
	"github.com/hortbot/hortbot/internal/pkg/docker/dpostgres"
	"github.com/peterldowns/pgtestdb"
	"gotest.tools/v3/assert"
)

// Pool provides access to a pool of fresh databases for testing.
// Its zero value is ready to use. If the pool is never used, then no
// database will be created.
type Pool struct {
	once sync.Once
	err  error

	pdb *dpostgres.DB
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

	info := p.pdb.Info()
	return pgtestdb.New(t, pgtestdb.Config{
		DriverName: info.DriverName,
		User:       info.User,
		Password:   info.Password,
		Host:       info.Host,
		Port:       info.Port,
		Database:   info.Database,
		Options:    info.Options,
	}, migrations.NewPGTestDBMigrator())
}
