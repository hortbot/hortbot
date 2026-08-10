// Package sqlflags processes SQL database related flags.
package sqlflags

import (
	"context"
	"time"

	"github.com/hortbot/hortbot/internal/db/migrations"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/zikaeroh/ctxlog"
	"go.uber.org/zap"
)

// SQL contains SQL database flags.
type SQL struct {
	DB        string `long:"db" env:"HB_DB" description:"PostgresSQL connection string" required:"true"`
	MigrateUp bool   `long:"db-migrate-up" env:"HB_DB_MIGRATE_UP" description:"Migrates the postgres database up"`
}

// Default contains the default flags. Make a copy of this, do not reuse.
var Default = SQL{}

// Open opens a PostgreSQL connection pool.
func (args *SQL) Open(ctx context.Context) *pgxpool.Pool {
	db, err := pgxpool.New(ctx, args.DB)
	if err != nil {
		ctxlog.Fatal(ctx, "error opening connection to database", zap.Error(err))
	}

	for {
		err := db.Ping(ctx)
		if err == nil {
			break
		}
		ctxlog.Warn(ctx, "error connecting to PostgreSQL; retrying", zap.Error(err))
		select {
		case <-time.After(time.Second):
		case <-ctx.Done():
			ctxlog.Fatal(ctx, "error connecting to database", zap.Error(ctx.Err()))
		}
	}

	if args.MigrateUp {
		if err := migrations.Up(args.DB, nil); err != nil {
			ctxlog.Fatal(ctx, "error migrating database", zap.Error(err))
		}
	}

	return db
}
