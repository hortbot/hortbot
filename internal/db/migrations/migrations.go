package migrations

import (
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/source"
	"github.com/golang-migrate/migrate/v4/source/httpfs"
	"github.com/hortbot/hortbot/internal/db/migrations/esc"

	_ "github.com/golang-migrate/migrate/v4/database/postgres" // golang-migrate postgres support
)

//go:generate gobin -m -run github.com/mjibson/esc -o=esc/esc.go -pkg=esc -ignore=esc -include=\.sql$ -modtime=0 .

var escSource source.Driver

func init() {
	var err error
	escSource, err = httpfs.New(esc.FS(false), "/")
	if err != nil {
		panic(err)
	}
}

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
	m, err := migrate.NewWithSourceInstance("esc", escSource, connStr)
	if err != nil {
		return nil, err
	}
	m.Log = logger

	return m, nil
}

func ignoreNoChange(err error) error {
	if err == migrate.ErrNoChange {
		return nil
	}
	return err
}

// LoggerFunc is a function that will be called with migration logging.
type LoggerFunc func(format string, v ...interface{})

// Printf implements migrate.Logger.
func (l LoggerFunc) Printf(format string, v ...interface{}) {
	if l != nil {
		l(format, v...)
	}
}

// Verbose implements migrate.Logger.
func (l LoggerFunc) Verbose() bool {
	return false
}
