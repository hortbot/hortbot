// Package sqlflags processes SQL database related flags.
package sqlflags

import (
	"context"
	"database/sql"
	"time"

	"github.com/hortbot/hortbot/internal/db/migrations"
	"github.com/hortbot/hortbot/internal/pkg/ctxlog"
	"go.uber.org/zap"

	_ "github.com/jackc/pgx/v4/stdlib" // For postgres.
)

type SQL struct {
	DB        string `long:"db" env:"HB_DB" description:"PostgresSQL connection string" required:"true"`
	MigrateUp bool   `long:"db-migrate-up" env:"HB_DB_MIGRATE_UP" description:"Migrates the postgres database up"`
}

var DefaultSQL = SQL{}

func (args *SQL) DriverName() string {
	return "pgx"
}

func (args *SQL) Open(ctx context.Context, driverName string) *sql.DB {
	db, err := sql.Open(driverName, args.DB)
	if err != nil {
		ctxlog.Fatal(ctx, "error opening connection to database", zap.Error(err))
	}

	if err := db.PingContext(ctx); err != nil {
		var perr error
		for i := 0; i < 4; i++ {
			time.Sleep(100 * time.Millisecond)

			if perr = db.PingContext(ctx); perr == nil {
				break
			}
		}
		if perr != nil {
			ctxlog.Fatal(ctx, "error pinging database", zap.Error(err))
		}
	}

	if args.MigrateUp {
		if err := migrations.Up(args.DB, nil); err != nil {
			ctxlog.Fatal(ctx, "error migrating database", zap.Error(err))
		}
	}

	return db
}
