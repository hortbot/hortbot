package pgpool_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/hortbot/hortbot/internal/db/dbsql"
	"github.com/hortbot/hortbot/internal/pkg/testpostgres/pgpool"
	"gotest.tools/v3/assert"
)

type debugLogger struct {
	messages []string
}

func (*debugLogger) Helper() {}

func (l *debugLogger) Logf(format string, args ...any) {
	l.messages = append(l.messages, fmt.Sprintf(format, args...))
}

func TestPool(t *testing.T) {
	t.Parallel()
	var pool pgpool.Pool
	t.Cleanup(pool.Cleanup)

	db := pool.FreshDB(t)
	assert.Assert(t, db != nil)
	defer db.Close()

	count, err := dbsql.New(db).CountChannels(t.Context())
	assert.NilError(t, err)
	assert.Equal(t, count, int64(0))

	logger := &debugLogger{}
	ctx := pgpool.WithDebug(t.Context(), logger)
	_, err = dbsql.New(db).CountChannels(ctx)
	assert.NilError(t, err)
	assert.Assert(t, len(logger.messages) > 0)
	assert.Assert(t, strings.Contains(strings.Join(logger.messages, "\n"), "SELECT COUNT(*) FROM channels"))
}

func TestPoolNoUse(t *testing.T) {
	t.Parallel()
	var pool pgpool.Pool
	pool.Cleanup()
}
