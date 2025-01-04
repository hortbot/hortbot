// Package dpostgres creates PostgreSQL database containers.
package dpostgres

import (
	"database/sql"
	"fmt"

	"github.com/hortbot/hortbot/internal/db/driver"
	"github.com/hortbot/hortbot/internal/pkg/docker"
)

const (
	user     = "postgres"
	password = "mysecretpassword"
	port     = "5432/tcp"
	database = "postgres"
	options  = "sslmode=disable"
)

type DB struct {
	container *docker.Container
}

func (d *DB) ConnStr() string {
	return "postgres://" + user + ":" + password + "@" + d.container.GetHostPort(port) + "/" + database + "?" + options
}

func (d *DB) Open() (*sql.DB, error) {
	return sql.Open(driver.Name, d.ConnStr()) //nolint:wrapcheck
}

func (d *DB) checkReady() error {
	db, err := d.Open()
	if err != nil {
		return fmt.Errorf("opening database: %w", err)
	}
	defer db.Close()

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
}

func (d *DB) Cleanup() {
	d.container.Cleanup()
}

// New creates and starts a fresh PostgreSQL server.
func New() (*DB, error) {
	return newDB()
}

func newDB() (*DB, error) {
	container := &docker.Container{
		Repository: "ghcr.io/zikaeroh/postgres-initialized",
		Tag:        "16",
		Cmd:        []string{"-F"},
		Ports:      []string{port},
		Ready: func(container *docker.Container) error {
			return (&DB{container: container}).checkReady()
		},
		ExpirySecs: 300,
	}

	if err := container.Start(); err != nil {
		return nil, fmt.Errorf("starting container: %w", err)
	}

	return &DB{container: container}, nil
}
