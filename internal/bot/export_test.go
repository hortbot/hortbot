package bot

import "context"

// Exports for testing.

var ErrBuiltinDisabled = errBuiltinDisabled

type (
	Session = session
)

const (
	LevelUnknown     = AccessLevelUnknown
	LevelEveryone    = AccessLevelEveryone
	LevelSubscriber  = AccessLevelSubscriber
	LevelModerator   = AccessLevelModerator
	LevelBroadcaster = AccessLevelBroadcaster
	LevelAdmin       = AccessLevelAdmin
)

func TestingBuiltin(name string, fn func(ctx context.Context, s *Session, cmd string, args string) error, minLevel AccessLevel) {
	if name == "" {
		panic("empty builtin name")
	}

	if fn == nil {
		panic("nil function")
	}

	if minLevel <= AccessLevelUnknown {
		panic("unknown level for added builtin " + name)
	}

	if _, ok := builtinCommands.m[name]; ok {
		panic(name + " already exists")
	}

	hf := handlerFunc{
		fn:       fn,
		minLevel: minLevel,
	}

	verifyHandlerMapEntry(name, hf)

	builtinCommands.m[name] = hf
}

func TestingAction(fn func(ctx context.Context, action string) (string, error, bool)) {
	testingAction = fn
}
