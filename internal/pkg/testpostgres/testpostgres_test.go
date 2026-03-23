package testpostgres_test

import (
	"testing"

	"github.com/hortbot/hortbot/internal/pkg/testpostgres"
	"gotest.tools/v3/assert"
)

func TestNew(t *testing.T) {
	t.Parallel()
	pdb, err := testpostgres.New()
	assert.NilError(t, err)
	defer pdb.Cleanup()
	assert.Assert(t, pdb.ConnStr() != "", "got connStr: %s", pdb.ConnStr())

	db, err := pdb.Open()
	assert.NilError(t, err)

	_, err = db.Query("SELECT count(*) FROM schema_migrations")
	assert.ErrorContains(t, err, "does not exist")
}
