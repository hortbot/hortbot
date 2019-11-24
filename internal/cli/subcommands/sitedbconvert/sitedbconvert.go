package sitedbconvert

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"github.com/go-sql-driver/mysql"
	"github.com/hortbot/hortbot/internal/cli"
	"github.com/hortbot/hortbot/internal/confimport/sitedb"
	"github.com/hortbot/hortbot/internal/pkg/ctxlog"
	"github.com/jmoiron/sqlx"
	"github.com/ory/dockertest/v3"
)

const Name = "site-db-convert"

const (
	filenameChannels = "site_channels.json"
	filenameVars     = "site_vars.json"
)

type cmd struct {
	cli.Common
	SiteDumps string `long:"site-dumps" description:"Directory containing coebot.tv database dumps, also where converted files will be placed" required:"true"`
}

func Run(args []string) {
	cli.Run(Name, args, &cmd{
		Common: cli.Common{
			Debug: true,
		},
	})
}

func (cmd *cmd) Main(ctx context.Context, _ []string) {
	outDir := filepath.Clean(cmd.SiteDumps)
	if d, err := os.Stat(outDir); err != nil {
		if os.IsNotExist(err) {
			ctxlog.Fatal(ctx, "output directory does not exist")
		}
		ctxlog.Fatal(ctx, "error stat-ing output directory", ctxlog.PlainError(err))
	} else if !d.IsDir() {
		ctxlog.Fatal(ctx, "output is not a directory")
	}

	pool, err := dockertest.NewPool("")
	if err != nil {
		ctxlog.Fatal(ctx, "error creating dockertest pool", ctxlog.PlainError(err))
	}

	const (
		password = "password"
		dbName   = "db"
	)

	resource, err := pool.RunWithOptions(&dockertest.RunOptions{
		Repository: "mariadb",
		Tag:        "10.1",
		Env:        []string{"MYSQL_ROOT_PASSWORD=" + password, "MYSQL_DATABASE=" + dbName},
		Mounts:     []string{cmd.SiteDumps + ":/docker-entrypoint-initdb.d"},
	})
	if err != nil {
		ctxlog.Fatal(ctx, "error creating MariaDB container", ctxlog.PlainError(err))
	}
	defer func() {
		if err := pool.Purge(resource); err != nil {
			ctxlog.Fatal(ctx, "error purging resource", ctxlog.PlainError(err))
		}
	}()

	if err := resource.Expire(uint((24 * time.Hour).Seconds())); err != nil {
		ctxlog.Fatal(ctx, "error setting container expiration", ctxlog.PlainError(err))
	}

	connStr := "root:" + password + "@tcp(" + resource.GetHostPort("3306/tcp") + ")/" + dbName

	var db *sqlx.DB

	pool.MaxWait = 5 * time.Minute
	err = pool.Retry(func() error {
		var err error
		db, err = sqlx.Open("mysql", connStr)
		if err != nil {
			return err
		}
		return db.Ping()
	})
	if err != nil {
		ctxlog.Fatal(ctx, "error waiting for database to be ready", ctxlog.PlainError(err))
	}
	defer db.Close()

	writeChannels(ctx, db, outDir)
	writeVars(ctx, db, outDir)
}

func writeChannels(ctx context.Context, db *sqlx.DB, outDir string) {
	channels, err := sitedb.Channels(ctx, db)
	if err != nil {
		ctxlog.Fatal(ctx, "error querying for channels", ctxlog.PlainError(err))
	}

	f, err := os.Create(filepath.Join(outDir, filenameChannels))
	if err != nil {
		ctxlog.Fatal(ctx, "error creating channels file", ctxlog.PlainError(err))
	}
	defer f.Close()

	if err := json.NewEncoder(f).Encode(channels); err != nil {
		ctxlog.Fatal(ctx, "error encoding channels", ctxlog.PlainError(err))
	}
}

func writeVars(ctx context.Context, db *sqlx.DB, outDir string) {
	vars, err := sitedb.Vars(ctx, db)
	if err != nil {
		ctxlog.Fatal(ctx, "error querying for vars", ctxlog.PlainError(err))
	}

	f, err := os.Create(filepath.Join(outDir, filenameVars))
	if err != nil {
		ctxlog.Fatal(ctx, "error creating vars file", ctxlog.PlainError(err))
	}
	defer f.Close()

	if err := json.NewEncoder(f).Encode(vars); err != nil {
		ctxlog.Fatal(ctx, "error encoding vars", ctxlog.PlainError(err))
	}
}

func init() {
	_ = mysql.SetLogger(noLog{})
}

type noLog struct{}

func (noLog) Print(v ...interface{}) {}
