package bot

import (
	"context"
	"errors"
)

var errNotImplemented = errors.New("not implemented")

type cmdFunc func(ctx context.Context, s *Session, args string) error

var builtins = map[string]cmdFunc{
	"command": cmdSimpleCommand,
	"bullet":  cmdBullet,
	"prefix":  cmdPrefix,
}
