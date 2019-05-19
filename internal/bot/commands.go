package bot

import (
	"context"
	"strings"
)

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
