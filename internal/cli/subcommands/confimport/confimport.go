// Package confimport implements the main command for the config dump importer.
package confimport

import (
	"context"
	"database/sql"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"github.com/hortbot/hortbot/internal/cli"
	"github.com/hortbot/hortbot/internal/cli/flags/sqlflags"
	"github.com/hortbot/hortbot/internal/confimport"
	"github.com/hortbot/hortbot/internal/pkg/ctxlog"
	"github.com/hortbot/hortbot/internal/pkg/dbx"
	"github.com/hortbot/hortbot/internal/pkg/jsonx"
)

type cmd struct {
	cli.Common
	SQL sqlflags.SQL

	Dir []string `long:"dir" description:"Directory of hortbot models"`

	Positional struct {
		Files []string `positional-arg-name:"FILE"`
	} `positional-args:"true"`
}

// Command returns a fresh conf-import command.
func Command() cli.Command {
	return &cmd{
		Common: cli.Common{
			Debug: true,
		},
		SQL: sqlflags.Default,
	}
}

func (*cmd) Name() string {
	return "conf-import"
}

func (c *cmd) Main(ctx context.Context, _ []string) {
	db := c.SQL.Open(ctx, c.SQL.DriverName())
	defer db.Close()

	ctx = ctxlog.WithOptions(ctx, ctxlog.NoTrace())

	todo := make([]string, 0, len(c.Positional.Files))

	for _, file := range c.Positional.Files {
		file = filepath.Clean(file)
		todo = append(todo, file)
	}

	for _, dir := range c.Dir {
		dir = filepath.Clean(dir)

		files, err := ioutil.ReadDir(dir)
		if err != nil {
			ctxlog.Fatal(ctx, "error reading dir", ctxlog.PlainError(err))
		}

		for _, file := range files {
			if ctx.Err() != nil {
				break
			}

			if file.IsDir() {
				continue
			}

			name := file.Name()

			if filepath.Ext(name) != ".json" {
				continue
			}

			filename := filepath.Join(dir, name)
			todo = append(todo, filename)
		}
	}

	if len(todo) == 0 {
		ctxlog.Fatal(ctx, "no files to import")
	}

	importOne := func(filename string) {
		select {
		case <-ctx.Done():
			return
		default:
		}

		f, err := os.Open(filename)
		if err != nil {
			ctxlog.Error(ctx, "error opening file", ctxlog.PlainError(err))
			return
		}
		defer f.Close()

		config := &confimport.Config{}

		if err := jsonx.DecodeSingle(f, config); err != nil {
			ctxlog.Error(ctx, "error parsing file", ctxlog.PlainError(err))
			return
		}

		err = dbx.Transact(ctx, db,
			dbx.SetLocalLockTimeout(5*time.Second),
			func(ctx context.Context, tx *sql.Tx) error {
				return config.Insert(ctx, tx)
			},
		)

		if err != nil {
			ctxlog.Error(ctx, "error inserting into database", ctxlog.PlainError(err))
		}
	}

	for _, filename := range todo {
		importOne(filename)
	}
}
