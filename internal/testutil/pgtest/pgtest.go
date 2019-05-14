package pgtest

import (
	"database/sql"
	"fmt"

	"github.com/hortbot/hortbot/internal/db/migrations"
	"github.com/ory/dockertest"

	_ "github.com/lib/pq" // For postgres.
)

func New() (db *sql.DB, close func(), retErr error) {
	return newDB(true)
}

func NewNoMigrate() (db *sql.DB, close func(), retErr error) {
	return newDB(false)
}

func newDB(doMigrate bool) (db *sql.DB, closer func(), retErr error) {
	pool, err := dockertest.NewPool("")
	if err != nil {
		return nil, nil, err
	}

	resource, err := pool.Run("zikaeroh/postgres-initialized", "latest", nil)
	if err != nil {
		return nil, nil, err
	}

	defer func() {
		if retErr != nil {
			pool.Purge(resource)
		}
	}()

	connStr := pgConnStr(resource.GetHostPort("5432/tcp"))

	err = pool.Retry(func() error {
		var err error
		db, err = sql.Open("postgres", connStr)
		if err != nil {
			return err
		}

		return db.Ping()
	})
	if err != nil {
		return nil, nil, err
	}

	defer func() {
		if retErr != nil {
			db.Close()
		}
	}()

	if doMigrate {
		if err := migrations.Up(db, nil); err != nil {
			return nil, nil, err
		}
	}

	return db, func() {
		db.Close()
		pool.Purge(resource)
	}, nil
}

func pgConnStr(addr string) string {
	return fmt.Sprintf(`postgres://postgres:mysecretpassword@%s/postgres?sslmode=disable`, addr)
}
