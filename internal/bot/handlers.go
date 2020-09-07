package bot

import (
	"context"
	"strings"

	"go.opencensus.io/trace"
)

type handlerMap struct {
	m          map[string]handlerFunc
	isBuiltins bool
}

func verifyHandlerMapEntry(name string, hf handlerFunc) {
	if name == "" {
		panic("empty name")
	}

	if name != strings.ToLower(name) {
		panic("name " + name + " is not lowercase")
	}

	if hf.fn == nil {
		panic("nil handler func")
	}

	if hf.minLevel == levelUnknown {
		panic("unknown minLevel")
	}
}

func newHandlerMap(m map[string]handlerFunc) handlerMap {
	for k, v := range m {
		verifyHandlerMapEntry(k, v)
	}
	return handlerMap{
		m: m,
	}
}

func (h handlerMap) Run(ctx context.Context, s *session, cmd string, args string) (bool, error) {
	return h.run(ctx, s, cmd, args, false)
}

func (h handlerMap) RunWithCooldown(ctx context.Context, s *session, cmd string, args string) (bool, error) {
	return h.run(ctx, s, cmd, args, true)
}

func (h handlerMap) run(ctx context.Context, s *session, cmd string, args string, checkCooldown bool) (bool, error) {
	cmd = strings.ToLower(cmd)

	ctx, span := trace.StartSpan(ctx, "handlerMap.run")
	defer span.End()

	span.AddAttributes(trace.StringAttribute("cmd", cmd))

	bc, ok := h.m[cmd]
	if !ok {
		return false, nil
	}

	if !s.UserLevel.CanAccess(bc.minLevel) {
		return true, errNotAuthorized
	}

	if checkCooldown && !bc.skipCooldown {
		if err := s.TryCooldown(ctx); err != nil {
			return false, err
		}
	}

	defer s.UsageContext(cmd)()

	return true, bc.fn(ctx, s, cmd, args)
}

type handlerFunc struct {
	fn           func(ctx context.Context, s *session, cmd string, args string) error
	minLevel     accessLevel
	skipCooldown bool
}
