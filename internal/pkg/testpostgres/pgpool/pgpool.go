// Package pgpool implements a pool of databases for testing.
package pgpool

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"

	"github.com/hortbot/hortbot/internal/db/migrations"
	"github.com/hortbot/hortbot/internal/pkg/testpostgres"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/tracelog"
	"github.com/peterldowns/pgtestdb"
	"gotest.tools/v3/assert"
)

// Pool provides access to a pool of fresh databases for testing.
// Its zero value is ready to use. If the pool is never used, then no
// database will be created.
type Pool struct {
	once sync.Once
	err  error

	pdb *testpostgres.DB
}

type debugLogger interface {
	Helper()
	Logf(format string, args ...any)
}

type debugLoggerKey struct{}

// WithDebug logs pgx queries executed with the returned context.
func WithDebug(ctx context.Context, logger debugLogger) context.Context {
	return context.WithValue(ctx, debugLoggerKey{}, logger)
}

type contextLogger struct{}

func (contextLogger) Log(ctx context.Context, _ tracelog.LogLevel, msg string, data map[string]any) {
	logger, ok := ctx.Value(debugLoggerKey{}).(debugLogger)
	if !ok {
		return
	}
	logger.Helper()
	logger.Logf("pgx %s: %v", msg, data)
}

func (p *Pool) init(t testing.TB) {
	t.Helper()

	p.once.Do(func() {
		p.err = func() error {
			var err error
			p.pdb, err = testpostgres.New()
			if err != nil {
				return fmt.Errorf("creating database: %w", err)
			}
			return nil
		}()
	})

	assert.NilError(t, p.err, "initializing database")
}

// Cleanup cleans up the pool. It's safe to run, even if the pool was never used.
func (p *Pool) Cleanup() {
	p.once.Do(func() {
		p.err = errors.New("pgpool: cleaned up")
	})

	if p.pdb != nil {
		p.pdb.Cleanup()
	}
}

// FreshDB creates a new database, migrated up.
func (p *Pool) FreshDB(t testing.TB) *pgxpool.Pool {
	t.Helper()
	p.init(t)

	info := p.pdb.Info()
	sqlDB := pgtestdb.New(t, pgtestdb.Config{
		DriverName: info.DriverName,
		User:       info.User,
		Password:   info.Password,
		Host:       info.Host,
		Port:       info.Port,
		Database:   info.Database,
		Options:    info.Options,
	}, migrations.NewPGTestDBMigrator())

	assert.NilError(t, sqlDB.QueryRowContext(t.Context(), "SELECT current_database()").Scan(&info.Database))
	config, err := pgxpool.ParseConfig(info.String())
	assert.NilError(t, err)
	config.ConnConfig.Tracer = &tracelog.TraceLog{
		Logger:   contextLogger{},
		LogLevel: tracelog.LogLevelInfo,
	}
	db, err := pgxpool.NewWithConfig(t.Context(), config)
	assert.NilError(t, err)
	t.Cleanup(db.Close)
	return db
}
