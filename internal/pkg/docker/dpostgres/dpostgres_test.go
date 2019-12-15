package dpostgres_test

import (
	"testing"

	"github.com/hortbot/hortbot/internal/pkg/docker/dpostgres"
	"github.com/jmoiron/sqlx"
	"gotest.tools/v3/assert"
)

func TestNew(t *testing.T) {
	if testing.Short() {
		t.Skip("requires starting a docker container")
	}

	db, connStr, cleanup, err := dpostgres.New()
	assert.NilError(t, err)
	assert.Assert(t, cleanup != nil)
	defer cleanup()
	assert.Assert(t, connStr != "", "got connStr: %s", connStr)
	assert.Assert(t, db != nil)

	dbx := sqlx.NewDb(db, "pgx")

	var count int
	err = dbx.Get(&count, "SELECT count(*) FROM schema_migrations")
	assert.NilError(t, err)
	assert.Equal(t, count, 1)
}

func TestNoMigrate(t *testing.T) {
	if testing.Short() {
		t.Skip("requires starting a docker container")
	}

	db, connStr, cleanup, err := dpostgres.NewNoMigrate()
	assert.NilError(t, err)
	assert.Assert(t, cleanup != nil)
	defer cleanup()
	assert.Assert(t, connStr != "", "got connStr: %s", connStr)
	assert.Assert(t, db != nil)

	dbx := sqlx.NewDb(db, "pgx")

	var count int
	err = dbx.Get(&count, "SELECT count(*) FROM schema_migrations")
	assert.ErrorContains(t, err, "does not exist")
	assert.Equal(t, count, 0)
}
