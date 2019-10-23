// Package cmdargs consolidates arguments and main function handling.
package cmdargs

import (
	"context"
	"fmt"
	"os"

	"github.com/hortbot/hortbot/internal/pkg/ctxlog"
	"github.com/hortbot/hortbot/internal/version"
	"github.com/jessevdk/go-flags"
	"github.com/joho/godotenv"
	"github.com/posener/ctxutil"
	"go.uber.org/zap"
)

type Common struct {
	Debug  bool                  `long:"debug" env:"HB_DEBUG" description:"Enables debug mode and the debug log level"`
	Config func(filename string) `long:"config" description:"A path to an INI config file - may be passed more than once"`
}

func (args *Common) debug() bool {
	return args.Debug
}

func (args *Common) configFunc(fn func(string)) {
	args.Config = fn
}

type Args interface {
	debug() bool
	configFunc(func(string))
}

var DefaultCommon = Common{}

func Run(args Args, main func(context.Context)) {
	ctx := ctxutil.Interrupt()
	_ = godotenv.Load()

	parser := flags.NewParser(args, flags.Default)
	args.configFunc(func(filename string) {
		err := flags.NewIniParser(parser).ParseFile(filename)
		checkParseError(err, true)
	})

	_, err := parser.Parse()
	checkParseError(err, false)

	logger := ctxlog.New(args.debug())
	defer zap.RedirectStdLog(logger)()
	ctx = ctxlog.WithLogger(ctx, logger)

	logger.Info("starting", zap.String("version", version.Version()))

	main(ctx)
}

func checkParseError(err error, print bool) {
	if err == nil {
		return
	}

	print = print && !flags.WroteHelp(err)

	if flagsErr, ok := err.(*flags.Error); ok && flagsErr.Type == flags.ErrHelp {
		if print {
			fmt.Fprintln(os.Stdout, err)
		}
		os.Exit(0)
	} else {
		if print {
			fmt.Fprintln(os.Stderr, err)
		}
		os.Exit(1)
	}
}
