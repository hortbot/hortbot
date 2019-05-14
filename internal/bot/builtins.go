package bot

import (
	"context"
	"errors"
)

var errNotImplemented = errors.New("not implemented")

var builtins = map[string]func(ctx context.Context, m *Message, args string) error{
	"command": command,
}

func command(ctx context.Context, m *Message, args string) error {
	return errNotImplemented
}
