package migrations_test

import (
	"database/sql"
	"fmt"
	"sort"
	"testing"

	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/hortbot/hortbot/internal/db/migrations"
	"github.com/ory/dockertest"
	"gotest.tools/assert"
	"gotest.tools/assert/cmp"
)

var allTables = []string{
	"schema_migrations",
	"channels",
	"simple_commands",
}

func TestUp(t *testing.T) {
	t.Parallel()

	withDatabase(t, func(t *testing.T, db *sql.DB) {
		assert.NilError(t, migrations.Up(db, t.Logf))
		assertTableNames(t, db, allTables...)
	})
}

func TestUpDown(t *testing.T) {
	t.Parallel()

	withDatabase(t, func(t *testing.T, db *sql.DB) {
		assert.NilError(t, migrations.Up(db, t.Logf))
		assert.NilError(t, migrations.Down(db, t.Logf))
		assertTableNames(t, db, "schema_migrations")
	})
}

func TestReset(t *testing.T) {
	t.Parallel()

	withDatabase(t, func(t *testing.T, db *sql.DB) {
		assert.NilError(t, migrations.Up(db, t.Logf))
		assertTableNames(t, db, allTables...)
		assert.NilError(t, migrations.Reset(db, t.Logf))
		assertTableNames(t, db, allTables...)
	})
}

func withDatabase(t *testing.T, fn func(t *testing.T, db *sql.DB)) {
	t.Helper()

	if testing.Short() {
		t.Skip("requires starting a docker container")
	}

	pool, err := dockertest.NewPool("")
	assert.NilError(t, err)

	resource, err := pool.Run("zikaeroh/postgres-initialized", "latest", nil)
	assert.NilError(t, err)

	var db *sql.DB
	err = pool.Retry(func() error {
		var err error
		db, err = sql.Open("postgres", connStr(resource.GetHostPort("5432/tcp")))
		if err != nil {
			return err
		}

		return db.Ping()
	})
	assert.NilError(t, err)

	defer func() {
		assert.NilError(t, pool.Purge(resource))
	}()

	assert.NilError(t, migrations.Up(db, nil))

	fn(t, db)
}

func connStr(addr string) string {
	return fmt.Sprintf(`postgres://postgres:mysecretpassword@%s/postgres?sslmode=disable`, addr)
}

func assertTableNames(t *testing.T, db *sql.DB, names ...string) {
	t.Helper()
	sort.Strings(names)

	tables := tableNames(t, db)
	sort.Strings(tables)

	assert.Check(t, cmp.DeepEqual(names, tables, cmpopts.EquateEmpty()))
}

func tableNames(t *testing.T, db *sql.DB) []string {
	t.Helper()

	query := `SELECT table_name FROM information_schema.tables WHERE table_schema=(SELECT current_schema()) AND table_type='BASE TABLE'`
	rows, err := db.Query(query)
	assert.NilError(t, err)
	defer rows.Close()

	var names []string
	for rows.Next() {
		var name string
		err := rows.Scan(&name)
		assert.NilError(t, err)
		if len(name) > 0 {
			names = append(names, name)
		}
	}

	return names
}
