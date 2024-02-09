package migrations_test

import (
	"database/sql"
	"sort"
	"testing"

	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/hortbot/hortbot/internal/db/migrations"
	"github.com/hortbot/hortbot/internal/pkg/docker/dpostgres"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"
)

// Returns a new slice for each invocation, so that each user can modify the list as needed.
func allTables() []string {
	return []string{
		"schema_migrations",
		"channels",
		"custom_commands",
		"quotes",
		"repeated_commands",
		"scheduled_commands",
		"autoreplies",
		"variables",
		"twitch_tokens",
		"blocked_users",
		"command_lists",
		"command_infos",
		"highlights",
	}
}

func TestUp(t *testing.T) {
	t.Parallel()

	withDatabase(t, func(t *testing.T, db *sql.DB, connStr string) {
		assert.NilError(t, migrations.Up(connStr, t.Logf))
		assertTableNames(t, db, allTables()...)
	})
}

func TestUpDown(t *testing.T) {
	t.Parallel()

	withDatabase(t, func(t *testing.T, db *sql.DB, connStr string) {
		assert.NilError(t, migrations.Up(connStr, t.Logf))
		assert.NilError(t, migrations.Down(connStr, t.Logf))
		assertTableNames(t, db, "schema_migrations")
	})
}

func TestReset(t *testing.T) {
	t.Parallel()

	withDatabase(t, func(t *testing.T, db *sql.DB, connStr string) {
		assert.NilError(t, migrations.Up(connStr, t.Logf))
		assertTableNames(t, db, allTables()...)
		assert.NilError(t, migrations.Reset(connStr, t.Logf))
		assertTableNames(t, db, allTables()...)
	})
}

func withDatabase(t *testing.T, fn func(t *testing.T, db *sql.DB, connStr string)) {
	t.Helper()

	db, connStr, cleanup, err := dpostgres.NewNoMigrate()
	assert.NilError(t, err, "creating new db without migrations")
	defer cleanup()

	fn(t, db, connStr)
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

	assert.NilError(t, rows.Err())

	return names
}

func TestBadConnStr(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		fn   func(string, migrations.LoggerFunc) error
	}{
		{name: "Up", fn: migrations.Up},
		{name: "Down", fn: migrations.Down},
		{name: "Reset", fn: migrations.Reset},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			assert.Error(t, test.fn(":", t.Logf), "no scheme")
		})
	}
}
