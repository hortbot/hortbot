package sqlflags

import (
	"context"
	"testing"
	"time"

	"github.com/hortbot/hortbot/internal/pkg/testpostgres"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
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
	opened := make(chan *pgxpool.Pool, 1)
	go func() {
		opened <- args.Open(ctx)
	}()

	select {
	case db := <-opened:
		db.Close()
		t.Fatal("database opened while PostgreSQL was stopped")
	case <-time.After(250 * time.Millisecond):
	}

	assert.NilError(t, pdb.Start())

	var db *pgxpool.Pool
	select {
	case db = <-opened:
	case <-ctx.Done():
		t.Fatal("database did not open after PostgreSQL started")
	}
	t.Cleanup(db.Close)

	var migrationsTable pgtype.Text
	assert.NilError(t, db.QueryRow(ctx,
		`SELECT to_regclass('public.schema_migrations')`).Scan(&migrationsTable))
	assert.Assert(t, migrationsTable.Valid)
}
