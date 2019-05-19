package bot

import "context"

// Exports for testing.

func Testing() {
	isTesting = true
}

func AddBuiltin(name string, fn func(ctx context.Context, s *Session, args string) error, minLevel AccessLevel) {
	if name == "" {
		panic("empty builtin name")
	}

	if fn == nil {
		panic("nil function")
	}

	if minLevel <= LevelUnknown {
		panic("unknown level for added builtin " + name)
	}

	builtins[name] = builtinCommand{
		fn:       fn,
		minLevel: minLevel,
	}
}
