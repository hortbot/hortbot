package dpostgres_test

import (
	"testing"

	"github.com/hortbot/hortbot/internal/pkg/docker/dpostgres"
	"github.com/jmoiron/sqlx"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/env"
)

func TestNew(t *testing.T) {
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

func TestNewBadDocker(t *testing.T) {
	defer env.Patch(t, "DOCKER_URL", "tcp://[[[[[")()

	_, _, _, err := dpostgres.New()
	assert.ErrorContains(t, err, "invalid endpoint")
}
