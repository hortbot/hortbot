// Package dpostgres creates PostgreSQL database containers.
package dpostgres

import (
	"database/sql"
	"fmt"

	"github.com/hortbot/hortbot/internal/db/driver"
	"github.com/hortbot/hortbot/internal/db/migrations"
	"github.com/hortbot/hortbot/internal/pkg/docker"
)

// New creates and starts a fresh PostgreSQL server, migrated up.
func New() (db *sql.DB, connStr string, cleanup func(), retErr error) {
	return newDB(true)
}

// NewNoMigrate creates and starts a fresh PostgreSQL server, without migrations.
func NewNoMigrate() (db *sql.DB, connStr string, cleanup func(), retErr error) {
	return newDB(false)
}

func newDB(doMigrate bool) (db *sql.DB, connStr string, cleanupr func(), retErr error) {
	container := &docker.Container{
		Repository: "ghcr.io/zikaeroh/postgres-initialized",
		Tag:        "16",
		Cmd:        []string{"-F"},
		Ready: func(container *docker.Container) error {
			connStr = "postgres://postgres:mysecretpassword@" + container.GetHostPort("5432/tcp") + "/postgres?sslmode=disable"

			var err error
			db, err = sql.Open(driver.Name, connStr)
			if err != nil {
				return fmt.Errorf("opening database: %w", err)
			}

			if err := db.Ping(); err != nil {
				return fmt.Errorf("pinging database: %w", err)
			}

			rows, err := db.Query("SELECT * FROM pg_catalog.pg_tables")
			if err != nil {
				return fmt.Errorf("querying database: %w", err)
			}
			if err := rows.Close(); err != nil {
				return fmt.Errorf("closing rows: %w", err)
			}
			if err := rows.Err(); err != nil {
				return fmt.Errorf("checking rows: %w", err)
			}

			return nil
		},
		ExpirySecs: 300,
	}

	if err := container.Start(); err != nil {
		return nil, "", nil, fmt.Errorf("starting container: %w", err)
	}

	if doMigrate {
		if err := migrations.Up(connStr, nil); err != nil {
			return nil, "", nil, fmt.Errorf("migrating database: %w", err)
		}
	}

	return db, connStr, func() {
		_ = db.Close()
		container.Cleanup()
	}, nil
}
