package confimport

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/hortbot/hortbot/internal/cli"
	"github.com/hortbot/hortbot/internal/cli/flags/sqlflags"
	"github.com/hortbot/hortbot/internal/confimport"
	"github.com/hortbot/hortbot/internal/pkg/ctxlog"
)

type cmd struct {
	cli.Common
	SQL sqlflags.SQL

	Dir   []string `long:"dir" description:"Directory of hortbot models"`
	Files []string `positional-args:""`
}

func Run(args []string) {
	cli.Run("conf-convert", args, &cmd{
		Common: cli.Common{
			Debug: true,
		},
		SQL: sqlflags.DefaultSQL,
	})
}

func (cmd *cmd) Main(ctx context.Context, _ []string) {
	logger := ctxlog.FromContext(ctx)

	connector := cmd.SQL.Connector(ctx)
	db := cmd.SQL.Open(ctx, connector)
	defer db.Close()

	logger = logger.WithOptions(ctxlog.NoTrace())
	ctx = ctxlog.WithLogger(ctx, logger)

	todo := make([]string, 0, len(cmd.Files))

	for _, file := range cmd.Files {
		file = filepath.Clean(file)
		todo = append(todo, file)
	}

	for _, dir := range cmd.Dir {
		dir = filepath.Clean(dir)

		files, err := ioutil.ReadDir(dir)
		if err != nil {
			logger.Fatal("error reading dir", ctxlog.PlainError(err))
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

	importOne := func(filename string) {
		f, err := os.Open(filename)
		if err != nil {
			logger.Error("error opening file", ctxlog.PlainError(err))
			return
		}
		defer f.Close()

		config := &confimport.Config{}

		if err := json.NewDecoder(f).Decode(config); err != nil {
			logger.Error("error parsing file", ctxlog.PlainError(err))
			return
		}

		if err := config.Insert(ctx, db); err != nil {
			logger.Error("error inserting into database", ctxlog.PlainError(err))
		}
	}

	for _, filename := range todo {
		importOne(filename)
	}
}
