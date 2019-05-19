package bot

import (
	"context"
)

type builtinCommand struct {
	fn       func(ctx context.Context, s *Session, args string) error
	minLevel AccessLevel
}

func (b builtinCommand) run(ctx context.Context, s *Session, args string) error {
	if !s.UserLevel.CanAccess(b.minLevel) {
		return errNotAuthorized
	}

	return b.fn(ctx, s, args)
}

var builtins = map[string]builtinCommand{
	"command": {fn: cmdSimpleCommand, minLevel: LevelModerator},
	"bullet":  {fn: cmdBullet, minLevel: LevelBroadcaster},
	"prefix":  {fn: cmdPrefix, minLevel: LevelBroadcaster},
}
