package sqlflags

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/hortbot/hortbot/internal/pkg/testpostgres"
	"gotest.tools/v3/assert"
)

func TestOpenWaitsForPostgres(t *testing.T) {
	t.Parallel()

	pdb, err := testpostgres.New()
	assert.NilError(t, err)
	t.Cleanup(pdb.Cleanup)
	assert.NilError(t, pdb.Stop())

	ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
	defer cancel()

	args := SQL{
		DB:        pdb.ConnStr(),
		MigrateUp: true,
	}
	opened := make(chan *sql.DB, 1)
	go func() {
		opened <- args.Open(ctx, args.DriverName())
	}()

	select {
	case db := <-opened:
		_ = db.Close()
		t.Fatal("database opened while PostgreSQL was stopped")
	case <-time.After(250 * time.Millisecond):
	}

	assert.NilError(t, pdb.Start())

	var db *sql.DB
	select {
	case db = <-opened:
	case <-ctx.Done():
		t.Fatal("database did not open after PostgreSQL started")
	}
	t.Cleanup(func() { assert.NilError(t, db.Close()) })

	var migrationsTable sql.NullString
	assert.NilError(t, db.QueryRowContext(ctx,
		`SELECT to_regclass('public.schema_migrations')`).Scan(&migrationsTable))
	assert.Assert(t, migrationsTable.Valid)
}
