// Package cli consolidates flag and main function handling.
package cli

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/hortbot/hortbot/internal/pkg/ctxlog"
	"github.com/hortbot/hortbot/internal/version"
	"github.com/jessevdk/go-flags"
	"github.com/joho/godotenv"
	"github.com/posener/ctxutil"
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
	Debug bool `long:"debug" env:"HB_DEBUG" description:"Enables debug mode and the debug log level"`
}

// IsDebug returns true if the command should be run in debug mode.
func (c *Common) IsDebug() bool {
	return c.Debug
}

// Command is a single command that can be parsed and run.
type Command interface {
	Main(ctx context.Context, args []string)
	IsDebug() bool
}

// Default contains the default flags. Make a copy of this, do not reuse.
var Default = Common{}

// Run parses the argument and runs the given command. If the ENV_FILE
// environment variable is set, the files listed in it will be loaded
// before parsing, to allow for a simple layered configuration setup.
func Run(name string, args []string, cmd Command) {
	_ = godotenv.Load(strings.Split(os.Getenv("ENV_FILE"), ",")...)

	ctx := ctxutil.Interrupt()

	parser := flags.NewNamedParser(name, flags.HelpFlag|flags.PassDoubleDash)
	_, _ = parser.AddGroup("Options", "", cmd)

	args, err := parser.ParseArgs(args)
	checkParseError(err)

	logger := ctxlog.New(cmd.IsDebug())
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
