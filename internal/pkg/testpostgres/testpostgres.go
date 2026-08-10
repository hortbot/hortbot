// Package testpostgres creates embedded PostgreSQL databases for testing.
package testpostgres

import (
	"context"
	"database/sql"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"

	embeddedpostgres "github.com/fergusstrange/embedded-postgres"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
)

const (
	user       = "postgres"
	password   = "postgres"
	database   = "postgres"
	options    = "sslmode=disable"
	driverName = "pgx"
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
	pg      *embeddedpostgres.EmbeddedPostgres
	port    uint32
	dataDir string
}

func (d *DB) Info() *Info {
	return &Info{
		DriverName: driverName,
		User:       user,
		Password:   password,
		Host:       "localhost",
		Port:       strconv.FormatUint(uint64(d.port), 10),
		Database:   database,
		Options:    options,
	}
}

func (d *DB) ConnStr() string {
	return d.Info().String()
}

func (d *DB) OpenSQL() (*sql.DB, error) {
	return sql.Open(driverName, d.ConnStr()) //nolint:wrapcheck
}

func (d *DB) Open(ctx context.Context) (*pgxpool.Pool, error) {
	return pgxpool.New(ctx, d.ConnStr()) //nolint:wrapcheck
}

// Stop stops the embedded PostgreSQL process without deleting its data.
func (d *DB) Stop() error {
	return d.pg.Stop() //nolint:wrapcheck
}

// Start restarts the embedded PostgreSQL process using its existing data.
func (d *DB) Start() error {
	return d.pg.Start() //nolint:wrapcheck
}

func (d *DB) Cleanup() {
	if d.pg != nil {
		_ = d.pg.Stop()
	}
	if d.dataDir != "" {
		_ = os.RemoveAll(d.dataDir)
	}
}

// New creates and starts a fresh PostgreSQL server using embedded-postgres.
func New() (*DB, error) {
	return newDB()
}

func newDB() (*DB, error) {
	const version = embeddedpostgres.V16

	port, err := freePort()
	if err != nil {
		return nil, fmt.Errorf("finding free port: %w", err)
	}

	cacheDir, err := os.UserCacheDir()
	if err != nil {
		return nil, fmt.Errorf("getting user cache dir: %w", err)
	}
	binariesPath := filepath.Join(cacheDir, "hortbot-testpostgres", string(version))

	dataDir, err := os.MkdirTemp("", "embedded-postgres-*")
	if err != nil {
		return nil, fmt.Errorf("creating temp dir: %w", err)
	}

	pg := embeddedpostgres.NewDatabase(embeddedpostgres.DefaultConfig().
		Version(version).
		Username(user).
		Password(password).
		Database(database).
		Port(port).
		RuntimePath(filepath.Join(dataDir, "runtime")).
		DataPath(filepath.Join(dataDir, "data")).
		BinariesPath(binariesPath).
		StartParameters(map[string]string{"fsync": "off"}).
		Logger(nil))

	if err := pg.Start(); err != nil {
		_ = os.RemoveAll(dataDir)
		return nil, fmt.Errorf("starting embedded postgres: %w", err)
	}

	return &DB{pg: pg, port: port, dataDir: dataDir}, nil
}

func freePort() (uint32, error) {
	// Use net to find a free port, then close the listener immediately.
	l, err := (&net.ListenConfig{}).Listen(context.Background(), "tcp", "127.0.0.1:0")
	if err != nil {
		return 0, fmt.Errorf("listening for free port: %w", err)
	}
	defer l.Close()
	//nolint:gosec
	return uint32(l.Addr().(*net.TCPAddr).Port), nil
}
