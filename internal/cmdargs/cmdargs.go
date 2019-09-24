// Package cmdargs consolidates arguments and main function handling.
package cmdargs

import (
	"context"
	"log"
	"os"

	"github.com/hortbot/hortbot/internal/pkg/ctxlog"
	"github.com/jessevdk/go-flags"
	"github.com/joho/godotenv"
	"github.com/posener/ctxutil"
	"go.uber.org/zap"
)

type Common struct {
	Debug bool `long:"debug" env:"HB_DEBUG" description:"Enables debug mode and the debug log level"`
}

func (args *Common) debug() bool {
	return args.Debug
}

type Args interface {
	debug() bool
}

var DefaultCommon = Common{}

func Run(args Args, main func(context.Context)) {
	ctx := ctxutil.Interrupt()
	_ = godotenv.Load()

	if _, err := flags.Parse(args); err != nil {
		if !flags.WroteHelp(err) {
			log.Fatalln(err)
		}
		os.Exit(1)
	}

	logger := ctxlog.New(args.debug())
	defer zap.RedirectStdLog(logger)()
	ctx = ctxlog.WithLogger(ctx, logger)

	main(ctx)
}
