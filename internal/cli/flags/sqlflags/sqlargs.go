// Package sqlflags processes SQL database related flags.
package sqlflags

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"time"

	"github.com/hortbot/hortbot/internal/db/migrations"
	"github.com/hortbot/hortbot/internal/pkg/ctxlog"
	"github.com/lib/pq"
	"go.uber.org/zap"
)

type SQL struct {
	DB        string `long:"db" env:"HB_DB" description:"PostgresSQL connection string" required:"true"`
	MigrateUp bool   `long:"migrate-up" env:"HB_MIGRATE_UP" description:"Migrates the postgres database up"`
}

var DefaultSQL = SQL{}

func (args *SQL) DBConnector(ctx context.Context) driver.Connector {
	connector, err := pq.NewConnector(args.DB)
	if err != nil {
		ctxlog.FromContext(ctx).Fatal("error creating postgres connector", zap.Error(err))
	}
	return connector
}

func (args *SQL) OpenDB(ctx context.Context, connector driver.Connector) *sql.DB {
	db := sql.OpenDB(connector)

	if err := db.PingContext(ctx); err != nil {
		var perr error
		for i := 0; i < 4; i++ {
			time.Sleep(100 * time.Millisecond)

			if perr = db.PingContext(ctx); perr == nil {
				break
			}
		}
		if perr != nil {
			ctxlog.FromContext(ctx).Fatal("error pinging database", zap.Error(err))
		}
	}

	if args.MigrateUp {
		if err := migrations.Up(args.DB, nil); err != nil {
			ctxlog.FromContext(ctx).Fatal("error migrating database", zap.Error(err))
		}
	}

	return db
}
