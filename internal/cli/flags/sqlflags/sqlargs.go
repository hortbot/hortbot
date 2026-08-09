// Package sqlflags processes SQL database related flags.
package sqlflags

import (
	"context"
	"database/sql"
	"time"

	"github.com/hortbot/hortbot/internal/db/driver"
	"github.com/hortbot/hortbot/internal/db/migrations"
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

// DriverName returns the default driver name to connect to the database.
func (args *SQL) DriverName() string {
	return driver.Name
}

// Open opens a database connection given the specified driver name.
func (args *SQL) Open(ctx context.Context, driverName string) *sql.DB {
	db, err := sql.Open(driverName, args.DB)
	if err != nil {
		ctxlog.Fatal(ctx, "error opening connection to database", zap.Error(err))
	}

	for {
		err := db.PingContext(ctx)
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
