package confimport

import (
	"context"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/hortbot/hortbot/internal/cli"
	"github.com/hortbot/hortbot/internal/cli/flags/sqlflags"
	"github.com/hortbot/hortbot/internal/confimport"
	"github.com/hortbot/hortbot/internal/pkg/ctxlog"
	"github.com/hortbot/hortbot/internal/pkg/jsonx"
)

const Name = "conf-import"

type cmd struct {
	cli.Common
	SQL sqlflags.SQL

	Dir []string `long:"dir" description:"Directory of hortbot models"`

	Positional struct {
		Files []string `positional-arg-name:"FILE"`
	} `positional-args:"true"`
}

func Run(args []string) {
	cli.Run(Name, args, &cmd{
		Common: cli.Common{
			Debug: true,
		},
		SQL: sqlflags.DefaultSQL,
	})
}

func (cmd *cmd) Main(ctx context.Context, _ []string) {
	db := cmd.SQL.Open(ctx, cmd.SQL.DriverName())
	defer db.Close()

	ctx = ctxlog.WithOptions(ctx, ctxlog.NoTrace())

	todo := make([]string, 0, len(cmd.Positional.Files))

	for _, file := range cmd.Positional.Files {
		file = filepath.Clean(file)
		todo = append(todo, file)
	}

	for _, dir := range cmd.Dir {
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

		if err := config.Insert(ctx, db); err != nil {
			ctxlog.Error(ctx, "error inserting into database", ctxlog.PlainError(err))
		}
	}

	for _, filename := range todo {
		importOne(filename)
	}
}
