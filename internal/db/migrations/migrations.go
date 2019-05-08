package migrations

import (
	"database/sql"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	bindata "github.com/golang-migrate/migrate/v4/source/go_bindata"
)

//go:generate gobin -m -run github.com/mjibson/esc -o=migrations.esc.go -pkg=migrations -ignore=\.go$ -private .

// assetNames provides a go-bindata like interface to use with esc until
// golang-migrate supports http.FileSystem.
func assetNames() []string {
	names := make([]string, 0, len(_escData))
	for name, entry := range _escData {
		if !entry.isDir {
			names = append(names, name[1:])
		}
	}
	return names
}

// Up brings the database up to date to the latest migration.
func Up(db *sql.DB, logger func(format string, v ...interface{})) error {
	m, err := newMigrate(db, logger)
	if err != nil {
		return err
	}

	return ignoreNoChange(m.Up())
}

// Down brings the database down by applying the down migrations.
func Down(db *sql.DB, logger func(format string, v ...interface{})) error {
	m, err := newMigrate(db, logger)
	if err != nil {
		return err
	}

	return ignoreNoChange(m.Down())
}

// Reset resets the database by bringing the database down, then up again.
func Reset(db *sql.DB, logger func(format string, v ...interface{})) error {
	if err := Down(db, logger); err != nil {
		return err
	}

	return Up(db, logger)
}

func newMigrate(db *sql.DB, logger func(format string, v ...interface{})) (*migrate.Migrate, error) {
	resource := bindata.Resource(assetNames(), func(name string) ([]byte, error) {
		return _escFSByte(false, "/"+name)
	})
	source, err := bindata.WithInstance(resource)
	if err != nil {
		return nil, err
	}

	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		return nil, err
	}

	m, err := migrate.NewWithInstance("go-bindata", source, "postgres", driver)
	if err != nil {
		return nil, err
	}
	m.Log = loggerFunc(logger)

	return m, nil
}

func ignoreNoChange(err error) error {
	if err == migrate.ErrNoChange {
		return nil
	}
	return err
}

type loggerFunc func(format string, v ...interface{})

func (l loggerFunc) Printf(format string, v ...interface{}) {
	if l != nil {
		l(format, v...)
	}
}

func (l loggerFunc) Verbose() bool {
	return false
}
