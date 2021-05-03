// Package cli consolidates flag and main function handling.
package cli

import (
	"context"
	"fmt"
	"net/http"
	_ "net/http/pprof" //nolint:gosec
	"os"
	"os/signal"
	"time"

	"github.com/hortbot/hortbot/internal/version"
	"github.com/jessevdk/go-flags"
	"github.com/zikaeroh/ctxlog"
	"go.uber.org/zap"
)

func init() {
	// Set a sane default.
	http.DefaultClient = &http.Client{
		Timeout: 15 * time.Second,
	}
}

// Common contains flags common to all commands.
type Common struct {
	Debug               bool   `long:"debug" env:"HB_DEBUG" description:"Enables debug mode and the debug log level"`
	DefaultServeMuxAddr string `long:"default-serve-mux-addr" env:"HB_DEFAULT_SERVE_MUX_ADDR" description:"An address to listen on for the default serve mux HTTP calls (pprof, etc)"`
}

// IsDebug returns true if the command should be run in debug mode.
func (c *Common) IsDebug() bool {
	return c.Debug
}

// RunDefaultServeMux runs the global default HTTP mux in the background, if
// the addr has been set.
func (c *Common) RunDefaultServeMux() {
	if c.DefaultServeMuxAddr != "" {
		go http.ListenAndServe(c.DefaultServeMuxAddr, nil) //nolint:errcheck
	}
}

// Command is a single command that can be parsed and run.
type Command interface {
	Name() string
	Main(ctx context.Context, args []string)
	IsDebug() bool
	RunDefaultServeMux()
}

// Default contains the default flags. Make a copy of this, do not reuse.
var Default = Common{}

// Run parses the argument and runs the given command. If the ENV_FILE
// environment variable is set, the files listed in it will be loaded
// before parsing, to allow for a simple layered configuration setup.
func Run(cmd Command, args []string) {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	parser := flags.NewNamedParser(cmd.Name(), flags.HelpFlag|flags.PassDoubleDash)
	_, _ = parser.AddGroup("Options", "", cmd)

	args, err := parser.ParseArgs(args)
	checkParseError(err)

	logger := ctxlog.New(cmd.IsDebug())
	defer zap.RedirectStdLog(logger)()
	ctx = ctxlog.WithLogger(ctx, logger)

	logger.Info("starting", zap.String("version", version.Version()))

	cmd.RunDefaultServeMux()
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
