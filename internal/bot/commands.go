package bot

import (
	"context"
)

type cmdFunc func(ctx context.Context, s *Session, args string) error

var builtins = map[string]cmdFunc{
	"command": cmdSimpleCommand,
	"bullet":  cmdBullet,
	"prefix":  cmdPrefix,
}
