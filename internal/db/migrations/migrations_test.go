package migrations_test

import (
	"database/sql"
	"slices"
	"testing"

	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/hortbot/hortbot/internal/db/migrations"
	"github.com/hortbot/hortbot/internal/pkg/testpostgres"
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
		"bot_action_usage_stats",
		"bot_autoreply_cooldowns",
		"bot_builtin_usage_stats",
		"bot_command_cooldowns",
		"bot_confirmations",
		"bot_filter_warnings",
		"bot_link_permits",
		"bot_raffle_entries",
		"bot_repeat_cooldowns",
		"bot_scheduled_command_cooldowns",
		"moderated_channels",
		"chat_message_queue_keys",
		"chat_message_queue",
		"eventsub_sync_requests",
		"web_auth_states",
	}
}

func TestUp(t *testing.T) {
	t.Parallel()

	withDatabase(t, func(t *testing.T, db *sql.DB, connStr string) {
		assert.NilError(t, migrations.Up(connStr, t.Logf))
		assertTableNames(t, db, allTables()...)
		assertBotStateTablesLogged(t, db)
		assert.NilError(t, migrations.Up(connStr, t.Logf))
		assertTableNames(t, db, allTables()...)
		assertBotStateTablesLogged(t, db)
	})
}

func TestUpDown(t *testing.T) {
	t.Parallel()

	withDatabase(t, func(t *testing.T, db *sql.DB, connStr string) {
		assert.NilError(t, migrations.Up(connStr, t.Logf))
		assert.NilError(t, migrations.Down(connStr, t.Logf))
		assertTableNames(t, db, "schema_migrations")
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

func TestResetBroken(t *testing.T) {
	t.Parallel()

	withDatabase(t, func(t *testing.T, db *sql.DB, connStr string) {
		_, err := db.ExecContext(t.Context(), `CREATE TABLE "schema_migrations" (bad text)`)
		assert.NilError(t, err)
		assert.ErrorContains(t, migrations.Reset(connStr, t.Logf), "schema_migrations")
	})
}

func withDatabase(t *testing.T, fn func(t *testing.T, db *sql.DB, connStr string)) {
	t.Helper()

	pdb, err := testpostgres.New()
	assert.NilError(t, err, "creating new db")

	db, err := pdb.OpenSQL()
	assert.NilError(t, err, "opening db")

	connStr := pdb.ConnStr()

	defer pdb.Cleanup()

	fn(t, db, connStr)
}

func assertTableNames(t *testing.T, db *sql.DB, names ...string) {
	t.Helper()
	slices.Sort(names)

	tables := tableNames(t, db)
	slices.Sort(tables)

	assert.Check(t, cmp.DeepEqual(names, tables, cmpopts.EquateEmpty()))
}

func tableNames(t *testing.T, db *sql.DB) []string {
	t.Helper()

	query := `SELECT table_name FROM information_schema.tables WHERE table_schema=(SELECT current_schema()) AND table_type='BASE TABLE'`
	rows, err := db.QueryContext(t.Context(), query)
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

func assertBotStateTablesLogged(t *testing.T, db *sql.DB) {
	t.Helper()

	rows, err := db.QueryContext(t.Context(), `
		SELECT relname
		FROM pg_class
		WHERE relnamespace = current_schema()::regnamespace
		  AND relkind = 'r'
		  AND relpersistence <> 'p'
		  AND (relname LIKE 'bot_%' OR relname = 'web_auth_states')
	`)
	assert.NilError(t, err)
	defer rows.Close()

	var unlogged []string
	for rows.Next() {
		var name string
		assert.NilError(t, rows.Scan(&name))
		unlogged = append(unlogged, name)
	}
	assert.NilError(t, rows.Err())
	assert.DeepEqual(t, unlogged, []string(nil))
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
			assert.ErrorContains(t, test.fn(":", t.Logf), "no scheme")
		})
	}
}
