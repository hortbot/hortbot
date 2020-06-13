// Package sitedbconvert implements the main command for the site-db dump converter.
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
	"github.com/hortbot/hortbot/internal/pkg/docker"
	"github.com/jmoiron/sqlx"
	"github.com/zikaeroh/ctxlog"
)

const (
	filenameChannels = "site_channels.json"
	filenameVars     = "site_vars.json"
)

type cmd struct {
	cli.Common
	SiteDumps string `long:"site-dumps" description:"Directory containing coebot.tv database dumps, also where converted files will be placed" required:"true"`
}

// Command returns a fresh site-db-convert command.
func Command() cli.Command {
	return &cmd{
		Common: cli.Common{
			Debug: true,
		},
	}
}

func (*cmd) Name() string {
	return "site-db-convert"
}

func (c *cmd) Main(ctx context.Context, _ []string) {
	outDir := filepath.Clean(c.SiteDumps)
	if d, err := os.Stat(outDir); err != nil {
		if os.IsNotExist(err) {
			ctxlog.Fatal(ctx, "output directory does not exist")
		}
		ctxlog.Fatal(ctx, "error stat-ing output directory", ctxlog.PlainError(err))
	} else if !d.IsDir() {
		ctxlog.Fatal(ctx, "output is not a directory")
	}

	const (
		password = "password"
		dbName   = "db"
	)

	var db *sqlx.DB
	container := &docker.Container{
		Repository: "mariadb",
		Tag:        "10.1",
		Env:        []string{"MYSQL_ROOT_PASSWORD=" + password, "MYSQL_DATABASE=" + dbName},
		Mounts:     []string{c.SiteDumps + ":/docker-entrypoint-initdb.d"},
		Ready: func(container *docker.Container) error {
			connStr := "root:" + password + "@tcp(" + container.GetHostPort("3306/tcp") + ")/" + dbName

			var err error
			db, err = sqlx.Open("mysql", connStr)
			if err != nil {
				return err
			}
			return db.Ping()
		},
		ReadyMaxWait: 5 * time.Minute,
		ExpirySecs:   uint((24 * time.Hour).Seconds()),
	}

	if err := container.Start(); err != nil {
		ctxlog.Fatal(ctx, "error creating database", ctxlog.PlainError(err))
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
