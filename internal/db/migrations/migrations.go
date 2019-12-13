package migrations

import (
	"database/sql"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	bindata "github.com/golang-migrate/migrate/v4/source/go_bindata"
	"github.com/hortbot/hortbot/internal/db/migrations/esc"

	_ "github.com/jackc/pgx/v4/stdlib" // For postgres.
)

//go:generate gobin -m -run github.com/mjibson/esc -o=esc/esc.go -pkg=esc -ignore=esc -include=\.sql$ -private -modtime=0 .

// Up brings the database up to date to the latest migration.
func Up(connStr string, logger func(format string, v ...interface{})) error {
	m, err := newMigrate(connStr, logger)
	if err != nil {
		return err
	}
	defer m.Close()

	return ignoreNoChange(m.Up())
}

// Down brings the database down by applying the down migrations.
func Down(connStr string, logger func(format string, v ...interface{})) error {
	m, err := newMigrate(connStr, logger)
	if err != nil {
		return err
	}
	defer m.Close()

	return ignoreNoChange(m.Down())
}

// Reset resets the database by bringing the database down, then up again.
func Reset(connStr string, logger func(format string, v ...interface{})) error {
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

func newMigrate(connStr string, logger func(format string, v ...interface{})) (*migrate.Migrate, error) {
	db, err := sql.Open("pgx", connStr)
	if err != nil {
		return nil, err
	}

	resource := bindata.Resource(esc.AssetNames(), esc.Asset)
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
