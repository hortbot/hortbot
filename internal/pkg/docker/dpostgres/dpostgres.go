// Package dpostgres creates PostgreSQL database containers.
package dpostgres

import (
	"database/sql"
	"fmt"
	"net"

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

type Info struct {
	DriverName string
	User       string
	Password   string
	Host       string
	Port       string
	Database   string
	Options    string
}

func (i *Info) String() string {
	return "postgres://" + i.User + ":" + i.Password + "@" + i.Host + ":" + i.Port + "/" + i.Database + "?" + i.Options
}

type DB struct {
	container *docker.Container
}

func (d *DB) Info() *Info {
	hostPort := d.container.GetHostPort(port)
	host, port, err := net.SplitHostPort(hostPort)
	if err != nil {
		panic(err)
	}

	return &Info{
		DriverName: driver.Name,
		User:       user,
		Password:   password,
		Host:       host,
		Port:       port,
		Database:   database,
		Options:    options,
	}
}

func (d *DB) ConnStr() string {
	return d.Info().String()
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
		Repository: "postgres",
		Tag:        "16",
		Cmd:        []string{"-F", "-c", "fsync=off"},
		Ports:      []string{port},
		Env:        []string{"POSTGRES_PASSWORD=" + password},
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
