package pgtest

import (
	"database/sql"
	"fmt"

	"github.com/hortbot/hortbot/internal/db/migrations"
	"github.com/ory/dockertest"

	_ "github.com/lib/pq" // For postgres.
)

func New() (db *sql.DB, connStr string, cleanup func(), retErr error) {
	return newDB(true)
}

func NewNoMigrate() (db *sql.DB, connStr string, cleanup func(), retErr error) {
	return newDB(false)
}

func newDB(doMigrate bool) (db *sql.DB, connStr string, cleanupr func(), retErr error) {
	pool, err := dockertest.NewPool("")
	if err != nil {
		return nil, "", nil, err
	}

	resource, err := pool.RunWithOptions(&dockertest.RunOptions{
		Repository: "zikaeroh/postgres-initialized",
		Tag:        "latest",
		Cmd:        []string{"-F"},
	})
	if err != nil {
		return nil, "", nil, err
	}

	defer func() {
		if retErr != nil {
			pool.Purge(resource) //nolint:errcheck
		}
	}()

	// Ensure the container is cleaned up, even if the process exits.
	if err := resource.Expire(300); err != nil {
		return nil, "", nil, err
	}

	connStr = pgConnStr(resource.GetHostPort("5432/tcp"))

	err = pool.Retry(func() error {
		var err error
		db, err = sql.Open("postgres", connStr)
		if err != nil {
			return err
		}

		return db.Ping()
	})
	if err != nil {
		return nil, "", nil, err
	}

	defer func() {
		if retErr != nil {
			db.Close()
		}
	}()

	if doMigrate {
		if err := migrations.Up(connStr, nil); err != nil {
			return nil, "", nil, err
		}
	}

	return db, connStr, func() {
		db.Close()
		pool.Purge(resource) //nolint:errcheck
	}, nil
}

func pgConnStr(addr string) string {
	return fmt.Sprintf(`postgres://postgres:mysecretpassword@%s/postgres?sslmode=disable`, addr)
}
