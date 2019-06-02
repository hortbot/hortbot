package bot

import (
	"context"
	"errors"
	"strings"
)

var errBuiltinDisabled = errors.New("bot: builtin disabled")

var builtinCommands builtinMap

var reservedCommandNames = map[string]bool{
	"builtin": true,
	"command": true,
	"set":     true,
}

func init() {
	// To prevent initialization loop.
	builtinCommands = map[string]builtinCommand{
		"command":     {fn: cmdSimpleCommand, minLevel: LevelModerator},
		"coemand":     {fn: cmdSimpleCommand, minLevel: LevelModerator},
		"set":         {fn: cmdSettings, minLevel: LevelModerator},
		"setting":     {fn: cmdSettings, minLevel: LevelModerator},
		"owner":       {fn: cmdOwnerModRegularIgnore, minLevel: LevelBroadcaster},
		"mod":         {fn: cmdOwnerModRegularIgnore, minLevel: LevelBroadcaster},
		"regular":     {fn: cmdOwnerModRegularIgnore, minLevel: LevelBroadcaster},
		"ignore":      {fn: cmdOwnerModRegularIgnore, minLevel: LevelModerator},
		"__roundtrip": {fn: cmdRoundtrip, minLevel: LevelAdmin},
	}
}

type builtinMap map[string]builtinCommand

func (b builtinMap) run(ctx context.Context, s *Session, cmd string, args string) (bool, error) {
	cmd = strings.ToLower(cmd)
	bc, ok := b[cmd]
	if !ok {
		return false, nil
	}

	return true, bc.run(ctx, s, cmd, args)
}

type builtinCommand struct {
	fn       func(ctx context.Context, s *Session, cmd string, args string) error
	minLevel AccessLevel
}

func (b builtinCommand) run(ctx context.Context, s *Session, cmd string, args string) error {
	if !s.UserLevel.CanAccess(b.minLevel) {
		return errNotAuthorized
	}

	return b.fn(ctx, s, cmd, args)
}

func tryBuiltinCommand(ctx context.Context, s *Session, cmd string, args string) (bool, error) {
	if cmd == "builtin" {
		cmd, args = splitSpace(args)
		cmd = strings.ToLower(cmd)
	}

	isBuiltin, err := builtinCommands.run(ctx, s, cmd, args)

	if err == errBuiltinDisabled {
		return true, nil
	}

	return isBuiltin, err
}
