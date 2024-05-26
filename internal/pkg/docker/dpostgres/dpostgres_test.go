package dpostgres_test

import (
	"testing"

	"github.com/hortbot/hortbot/internal/pkg/docker/dpostgres"
	"gotest.tools/v3/assert"
)

func TestNew(t *testing.T) {
	db, connStr, cleanup, err := dpostgres.New()
	assert.NilError(t, err)
	assert.Assert(t, cleanup != nil)
	defer cleanup()
	assert.Assert(t, connStr != "", "got connStr: %s", connStr)
	assert.Assert(t, db != nil)

	rows, err := db.Query("SELECT count(*) FROM schema_migrations")
	assert.NilError(t, err)

	var count int
	assert.Assert(t, rows.Next())
	assert.NilError(t, rows.Scan(&count))
	assert.NilError(t, rows.Close())
	assert.Equal(t, count, 1)
}

func TestNoMigrate(t *testing.T) {
	db, connStr, cleanup, err := dpostgres.NewNoMigrate()
	assert.NilError(t, err)
	assert.Assert(t, cleanup != nil)
	defer cleanup()
	assert.Assert(t, connStr != "", "got connStr: %s", connStr)
	assert.Assert(t, db != nil)

	_, err = db.Query("SELECT count(*) FROM schema_migrations")
	assert.ErrorContains(t, err, "does not exist")
}

func TestNewBadDocker(t *testing.T) {
	t.Setenv("DOCKER_URL", "tcp://[[[[[")

	_, _, _, err := dpostgres.New()
	assert.ErrorContains(t, err, "invalid endpoint")
}
