package bot

import "context"

// Exports for testing.

func Testing() {
	isTesting = true
}

func TestingBuiltin(name string, fn func(ctx context.Context, s *Session, cmd string, args string) error, minLevel AccessLevel) {
	if name == "" {
		panic("empty builtin name")
	}

	if fn == nil {
		panic("nil function")
	}

	if minLevel <= LevelUnknown {
		panic("unknown level for added builtin " + name)
	}

	if _, ok := builtinCommands[name]; ok {
		panic(name + " already exists")
	}

	builtinCommands[name] = builtinCommand{
		fn:       fn,
		minLevel: minLevel,
	}
}

func TestingAction(fn func(ctx context.Context, action string) (string, error, bool)) {
	testingAction = fn
}

var ErrBuiltinDisabled = errBuiltinDisabled
