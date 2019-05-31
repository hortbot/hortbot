package bot

import (
	"context"
	"errors"
	"strings"

	"github.com/hortbot/hortbot/internal/ctxlog"
	"go.uber.org/zap"
)

var errBuiltinDisabled = errors.New("bot: builtin disabled")

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

var builtinCommands map[string]builtinCommand

func init() {
	// To prevent initialization loop.

	builtinCommands = map[string]builtinCommand{
		"command":     {fn: cmdSimpleCommand, minLevel: LevelModerator},
		"coemand":     {fn: cmdSimpleCommand, minLevel: LevelModerator},
		"bullet":      {fn: cmdBullet, minLevel: LevelBroadcaster},
		"prefix":      {fn: cmdPrefix, minLevel: LevelBroadcaster},
		"owner":       {fn: cmdOwnerModRegularIgnore, minLevel: LevelBroadcaster},
		"mod":         {fn: cmdOwnerModRegularIgnore, minLevel: LevelBroadcaster},
		"regular":     {fn: cmdOwnerModRegularIgnore, minLevel: LevelBroadcaster},
		"ignore":      {fn: cmdOwnerModRegularIgnore, minLevel: LevelModerator},
		"__roundtrip": {fn: cmdRoundtrip, minLevel: LevelAdmin},
	}
}

var reservedCommandNames = map[string]bool{
	"builtin": true,
	"command": true,
}

func tryBuiltinCommand(ctx context.Context, s *Session, name string, params string) (bool, error) {
	builtin := name == "builtin"
	if builtin {
		name, params = splitSpace(params)
		name = strings.ToLower(name)
	}

	if bc, ok := builtinCommands[name]; ok {
		err := bc.run(ctx, s, name, params)

		switch err {
		case errNotAuthorized:
			logger := ctxlog.FromContext(ctx)
			logger.Debug("error in builtin command", zap.Error(err))

		case errBuiltinDisabled:
			return false, nil
		}

		return true, err
	}

	return builtin, nil
}
