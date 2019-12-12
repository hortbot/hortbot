package pgtest

import (
	"database/sql"

	"github.com/hortbot/hortbot/internal/db/migrations"
	"github.com/hortbot/hortbot/internal/pkg/docker"

	_ "github.com/jackc/pgx/v4/stdlib" // For postgres.
)

func New() (db *sql.DB, connStr string, cleanup func(), retErr error) {
	return newDB(true)
}

func NewNoMigrate() (db *sql.DB, connStr string, cleanup func(), retErr error) {
	return newDB(false)
}

func newDB(doMigrate bool) (db *sql.DB, connStr string, cleanupr func(), retErr error) {
	container := &docker.Container{
		Repository: "zikaeroh/postgres-initialized",
		Tag:        "latest",
		Cmd:        []string{"-F"},
		Ready: func(container *docker.Container) error {
			connStr = "postgres://postgres:mysecretpassword@" + container.GetHostPort("5432/tcp") + "/postgres?sslmode=disable"

			var err error
			db, err = sql.Open("pgx", connStr)
			if err != nil {
				return err
			}

			return db.Ping()
		},
		ExpirySecs: 300,
	}

	if err := container.Start(); err != nil {
		return nil, "", nil, err
	}

	if doMigrate {
		if err := migrations.Up(connStr, nil); err != nil {
			return nil, "", nil, err
		}
	}

	return db, connStr, func() {
		_ = db.Close()
		container.Cleanup()
	}, nil
}
