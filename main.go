package main

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/hortbot/hortbot/internal/cli/subcommands/bot"
	"github.com/hortbot/hortbot/internal/cli/subcommands/confconvert"
	"github.com/hortbot/hortbot/internal/cli/subcommands/confimport"
	"github.com/hortbot/hortbot/internal/cli/subcommands/irc"
	"github.com/hortbot/hortbot/internal/cli/subcommands/singleproc"
	"github.com/hortbot/hortbot/internal/cli/subcommands/web"
	"github.com/hortbot/hortbot/internal/version"
)

var subcommands = map[string]func([]string){
	"bot":          bot.Run,
	"conf-convert": confconvert.Run,
	"conf-import":  confimport.Run,
	"irc":          irc.Run,
	"single-proc":  singleproc.Run,
	"web":          web.Run,
	"version":      func([]string) { fmt.Println("hortbot", version.Version()) },
}

func main() {
	args := os.Args[1:]

	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "Please specify a subcommand.")
		listAndExit()
	}

	subcommand, args := args[0], args[1:]
	if fn := subcommands[subcommand]; fn != nil {
		fn(args)
	} else {
		switch subcommand {
		case "-h", "--help":
		default:
			fmt.Fprintln(os.Stderr, subcommand, "is not a valid subcommand.")
		}
		listAndExit()
	}
}

func listAndExit() {
	names := make([]string, 0, len(subcommands))

	for name := range subcommands {
		names = append(names, name)
	}

	sort.Strings(names)

	fmt.Fprintln(os.Stderr, "Available subcommands:", strings.Join(names, ", "))
	os.Exit(2)
}
