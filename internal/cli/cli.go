// Package cli consolidates flag and main function handling.
package cli

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/hortbot/hortbot/internal/pkg/ctxlog"
	"github.com/hortbot/hortbot/internal/version"
	"github.com/jessevdk/go-flags"
	"github.com/posener/ctxutil"
	"go.uber.org/zap"
)

func init() {
	// Set a sane default.
	http.DefaultClient = &http.Client{
		Timeout: 15 * time.Second,
	}
}

type Common struct {
	Debug  bool                  `long:"debug" env:"HB_DEBUG" description:"Enables debug mode and the debug log level"`
	Config func(filename string) `short:"c" long:"config" description:"A path to an INI config file - may be passed more than once"`
}

func (c *Common) debug() bool {
	return c.Debug
}

func (c *Common) configFunc(fn func(string)) {
	c.Config = fn
}

type Command interface {
	Main(ctx context.Context, args []string)
	debug() bool
	configFunc(func(string))
}

var DefaultCommon = Common{}

func Run(name string, args []string, cmd Command) {
	ctx := ctxutil.Interrupt()

	parser := flags.NewNamedParser(name, flags.HelpFlag|flags.PassDoubleDash)
	_, _ = parser.AddGroup("Application Options", "", cmd)

	cmd.configFunc(func(filename string) {
		err := flags.NewIniParser(parser).ParseFile(filename)
		checkParseError(err)
	})

	args, err := parser.Parse()
	checkParseError(err)

	logger := ctxlog.New(cmd.debug())
	defer zap.RedirectStdLog(logger)()
	ctx = ctxlog.WithLogger(ctx, logger)

	logger.Info("starting", zap.String("version", version.Version()))

	cmd.Main(ctx, args)
}

func checkParseError(err error) {
	if err == nil {
		return
	}

	if flagsErr, ok := err.(*flags.Error); ok && flagsErr.Type == flags.ErrHelp {
		fmt.Fprintln(os.Stdout, err)
		os.Exit(0)
	} else {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
