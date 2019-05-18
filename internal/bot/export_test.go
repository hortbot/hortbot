package bot

import "context"

func AddBuiltin(name string, fn func(ctx context.Context, s *Session, args string) error) {
	builtins[name] = fn
}
