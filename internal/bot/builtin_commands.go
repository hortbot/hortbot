package bot

import (
	"context"
	"strings"
)

var builtinCommands handlerMap

var reservedCommandNames = map[string]bool{
	"builtin": true,
	"command": true,
	"set":     true,
}

func init() {
	// To prevent initialization loop.
	builtinCommands = map[string]handlerFunc{
		"command":     {fn: cmdSimpleCommand, minLevel: LevelModerator},
		"coemand":     {fn: cmdSimpleCommand, minLevel: LevelModerator},
		"set":         {fn: cmdSettings, minLevel: LevelModerator},
		"setting":     {fn: cmdSettings, minLevel: LevelModerator},
		"owner":       {fn: cmdOwnerModRegularIgnore, minLevel: LevelBroadcaster},
		"mod":         {fn: cmdOwnerModRegularIgnore, minLevel: LevelBroadcaster},
		"regular":     {fn: cmdOwnerModRegularIgnore, minLevel: LevelBroadcaster},
		"ignore":      {fn: cmdOwnerModRegularIgnore, minLevel: LevelModerator},
		"quote":       {fn: cmdQuote, minLevel: LevelSubscriber},
		"clear":       {fn: cmdModClear, minLevel: LevelModerator},
		"filter":      {fn: cmdFilter, minLevel: LevelModerator},
		"__roundtrip": {fn: cmdRoundtrip, minLevel: LevelAdmin},
	}
}

type handlerMap map[string]handlerFunc

func (h handlerMap) run(ctx context.Context, s *Session, cmd string, args string) (bool, error) {
	cmd = strings.ToLower(cmd)
	bc, ok := h[cmd]
	if !ok {
		return false, nil
	}

	defer s.UsageContext(cmd)()

	return true, bc.run(ctx, s, cmd, args)
}

type handlerFunc struct {
	fn       func(ctx context.Context, s *Session, cmd string, args string) error
	minLevel AccessLevel
}

func (h handlerFunc) run(ctx context.Context, s *Session, cmd string, args string) error {
	if !s.UserLevel.CanAccess(h.minLevel) {
		return errNotAuthorized
	}

	return h.fn(ctx, s, cmd, args)
}
