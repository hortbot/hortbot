package bot

import (
	"context"
	"strings"
)

type builtinCommand struct {
	minLevel int
	fn       func(ctx context.Context, s *Session, args string) error
}

func (b builtinCommand) run(ctx context.Context, s *Session, args string) error {
	if b.minLevel == 0 {
		return errNotAuthorized
	}

	return b.fn(ctx, s, args)
}

type cmdFunc func(ctx context.Context, s *Session, args string) error

var builtins = map[string]cmdFunc{
	"command": cmdSimpleCommand,
	"bullet":  cmdBullet,
	"prefix":  cmdPrefix,
}

func splitSpace(args string) (string, string) {
	s := strings.SplitN(args, " ", 2)

	switch len(s) {
	case 0:
		return "", ""
	case 1:
		return s[0], ""
	default:
		return s[0], strings.TrimSpace(s[1])
	}
}
