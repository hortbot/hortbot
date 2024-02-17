package main

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/hortbot/hortbot/internal/cli"
	"github.com/hortbot/hortbot/internal/cli/subcommands/bot"
	"github.com/hortbot/hortbot/internal/cli/subcommands/confconvert"
	"github.com/hortbot/hortbot/internal/cli/subcommands/confimport"
	"github.com/hortbot/hortbot/internal/cli/subcommands/irc"
	"github.com/hortbot/hortbot/internal/cli/subcommands/web"
	"github.com/hortbot/hortbot/internal/version"
	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load(strings.Split(os.Getenv("ENV_FILE"), ",")...)

	args := os.Args[1:]

	subcommands := make(map[string]cli.Command)
	addCommand := func(cmd cli.Command) {
		name := cmd.Name()
		if subcommands[name] != nil {
			panic("duplicate command " + name)
		}
		subcommands[name] = cmd
	}

	listAndExit := func() {
		names := make([]string, 0, len(subcommands))

		for name := range subcommands {
			names = append(names, name)
		}

		sort.Strings(names)

		fmt.Fprintln(os.Stderr, "Available subcommands:", strings.Join(names, ", "))
		os.Exit(2)
	}

	addCommand(bot.Command())
	addCommand(irc.Command())
	addCommand(web.Command())
	addCommand(confconvert.Command())
	addCommand(confimport.Command())

	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "Please specify a subcommand.")
		listAndExit()
	}

	subcommand, args := args[0], args[1:]

	if subcommand == "version" {
		fmt.Println(version.Version())
		return
	}

	if cmd := subcommands[subcommand]; cmd != nil {
		cli.Run(cmd, args)
		return
	}

	switch subcommand {
	case "-h", "--help":
	default:
		fmt.Fprintln(os.Stderr, subcommand, "is not a valid subcommand.")
	}
	listAndExit()
}
