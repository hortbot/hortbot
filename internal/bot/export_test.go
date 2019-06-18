package bot

import "context"

// Exports for testing.

var ErrBuiltinDisabled = errBuiltinDisabled

type Session = session
type AccessLevel = accessLevel

const (
	LevelUnknown     = levelUnknown
	LevelEveryone    = levelEveryone
	LevelSubscriber  = levelSubscriber
	LevelModerator   = levelModerator
	LevelBroadcaster = levelBroadcaster
	LevelAdmin       = levelAdmin
)

func Testing() {
	isTesting = true
}

func TestingBuiltin(name string, fn func(ctx context.Context, s *session, cmd string, args string) error, minLevel AccessLevel) {
	if name == "" {
		panic("empty builtin name")
	}

	if fn == nil {
		panic("nil function")
	}

	if minLevel <= levelUnknown {
		panic("unknown level for added builtin " + name)
	}

	if _, ok := builtinCommands[name]; ok {
		panic(name + " already exists")
	}

	builtinCommands[name] = handlerFunc{
		fn:       fn,
		minLevel: minLevel,
	}
}

func TestingAction(fn func(ctx context.Context, action string) (string, error, bool)) {
	testingAction = fn
}
