// Package migrations performs database migrations embedded in the binary.
package migrations

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"sync"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres" // golang-migrate postgres support
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/hortbot/hortbot/internal/pkg/must"
	"github.com/peterldowns/pgtestdb"
	"github.com/peterldowns/pgtestdb/migrators/common"
)

//go:embed static
var static embed.FS

var sourceDriver = must.Must(iofs.New(static, "static"))

// Up brings the database up to date to the latest migration.
func Up(connStr string, logger LoggerFunc) error {
	m, err := newMigrate(connStr, logger)
	if err != nil {
		return err
	}
	defer m.Close()

	return ignoreNoChange(m.Up())
}

// Down brings the database down by applying the down migrations.
func Down(connStr string, logger LoggerFunc) error {
	m, err := newMigrate(connStr, logger)
	if err != nil {
		return err
	}
	defer m.Close()

	return ignoreNoChange(m.Down())
}

// Reset resets the database by bringing the database down, then up again.
func Reset(connStr string, logger LoggerFunc) error {
	m, err := newMigrate(connStr, logger)
	if err != nil {
		return err
	}
	defer m.Close()

	if err := ignoreNoChange(m.Down()); err != nil {
		return err
	}

	return ignoreNoChange(m.Up())
}

func newMigrate(connStr string, logger LoggerFunc) (*migrate.Migrate, error) {
	m, err := migrate.NewWithSourceInstance("iofs", sourceDriver, connStr)
	if err != nil {
		return nil, fmt.Errorf("creating migrate instance: %w", err)
	}
	m.Log = logger

	return m, nil
}

func ignoreNoChange(err error) error {
	if errors.Is(err, migrate.ErrNoChange) {
		return nil
	}
	return err
}

// LoggerFunc is a function that will be called with migration logging.
type LoggerFunc func(format string, v ...any)

// Printf implements migrate.Logger.
func (l LoggerFunc) Printf(format string, v ...any) {
	if l != nil {
		l(format, v...)
	}
}

// Verbose implements migrate.Logger.
func (l LoggerFunc) Verbose() bool {
	return false
}

var hashDirs = sync.OnceValues(func() (string, error) {
	return common.HashDirs(static, "*.sql", "static")
})

func NewPGTestDBMigrator() pgtestdb.Migrator {
	return &pgtestdbMigrator{}
}

type pgtestdbMigrator struct{}

func (p *pgtestdbMigrator) Hash() (string, error) {
	return hashDirs()
}

func (*pgtestdbMigrator) Migrate(_ context.Context, _ *sql.DB, c pgtestdb.Config) error {
	return Up(c.URL(), nil)
}
