package bot

import (
	"context"
	"strconv"
	"strings"
)

var varCommands = newHandlerMap(map[string]handlerFunc{
	"set":       {fn: cmdVarSet, minLevel: AccessLevelModerator},
	"get":       {fn: cmdVarGet, minLevel: AccessLevelModerator},
	"delete":    {fn: cmdVarDelete, minLevel: AccessLevelModerator},
	"remove":    {fn: cmdVarDelete, minLevel: AccessLevelModerator},
	"increment": {fn: cmdVarIncrement, minLevel: AccessLevelModerator},
	"decrement": {fn: cmdVarIncrement, minLevel: AccessLevelModerator},
})

func cmdVar(ctx context.Context, s *session, cmd string, args string) error {
	subcommand, args := splitSpace(args)
	subcommand = strings.ToLower(subcommand)

	ok, err := varCommands.Run(ctx, s, subcommand, args)
	if ok || err != nil {
		return err
	}

	if !ok {
		return s.ReplyUsage(ctx, "set|get|delete|increment|decrement")
	}

	return nil
}

func cmdVarSet(ctx context.Context, s *session, cmd string, args string) error {
	name, value := splitSpace(args)

	if name == "" || value == "" {
		return s.ReplyUsage(ctx, "<name> <value>")
	}

	if err := s.VarSet(ctx, name, value); err != nil {
		return err
	}

	return s.Replyf(ctx, "Variable %s set to: %s", name, value)
}

func cmdVarGet(ctx context.Context, s *session, cmd string, args string) error {
	if args == "" {
		return s.ReplyUsage(ctx, "<name>")
	}

	value, ok, err := s.VarGet(ctx, args)
	if err != nil {
		return err
	}

	if !ok {
		return s.Replyf(ctx, "Variable %s does not exist.", args)
	}

	return s.Replyf(ctx, "Variable %s is set to: %s", args, value)
}

func cmdVarDelete(ctx context.Context, s *session, cmd string, args string) error {
	if args == "" {
		return s.ReplyUsage(ctx, "<name>")
	}

	if err := s.VarDelete(ctx, args); err != nil {
		return err
	}

	return s.Replyf(ctx, "Variable %s has been deleted.", args)
}

func cmdVarIncrement(ctx context.Context, s *session, cmd string, args string) error {
	usage := func() error {
		return s.ReplyUsage(ctx, "<name> <value>")
	}

	name, value := splitSpace(args)
	if name == "" {
		return usage()
	}

	inc, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return usage()
	}

	if cmd == "decrement" {
		inc = 0 - inc
	}

	x, badVar, err := s.VarIncrement(ctx, name, inc)

	switch {
	case err != nil:
		return err
	case badVar:
		return s.Replyf(ctx, "Variable %s is not an integer.", name)
	default:
		return s.Replyf(ctx, "Variable %s has been %sed to %d.", name, cmd, x)
	}
}
