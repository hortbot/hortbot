package bot

import (
	"context"
	"strings"
)

var builtinCommands handlerMap

var reservedCommandNames = map[string]bool{
	"builtin": true,
	"command": true,
	"set":     true,
}

func init() {
	// To prevent initialization loop.
	builtinCommands = map[string]handlerFunc{
		"command":     {fn: cmdSimpleCommand, minLevel: levelModerator},
		"coemand":     {fn: cmdSimpleCommand, minLevel: levelModerator},
		"set":         {fn: cmdSettings, minLevel: levelModerator},
		"setting":     {fn: cmdSettings, minLevel: levelModerator},
		"owner":       {fn: cmdOwnerModRegularIgnore, minLevel: levelBroadcaster},
		"mod":         {fn: cmdOwnerModRegularIgnore, minLevel: levelBroadcaster},
		"regular":     {fn: cmdOwnerModRegularIgnore, minLevel: levelBroadcaster},
		"ignore":      {fn: cmdOwnerModRegularIgnore, minLevel: levelModerator},
		"quote":       {fn: cmdQuote, minLevel: levelSubscriber},
		"clear":       {fn: cmdModClear, minLevel: levelModerator},
		"filter":      {fn: cmdFilter, minLevel: levelModerator},
		"permit":      {fn: cmdPermit, minLevel: levelModerator},
		"allow":       {fn: cmdPermit, minLevel: levelModerator},
		"leave":       {fn: cmdLeave, minLevel: levelBroadcaster},
		"part":        {fn: cmdLeave, minLevel: levelBroadcaster},
		"conch":       {fn: cmdConch, minLevel: levelSubscriber},
		"helix":       {fn: cmdConch, minLevel: levelSubscriber},
		"repeat":      {fn: cmdRepeat, minLevel: levelModerator},
		"schedule":    {fn: cmdSchedule, minLevel: levelModerator},
		"lastfm":      {fn: cmdLastFM, minLevel: levelEveryone},
		"songlink":    {fn: cmdSonglink, minLevel: levelEveryone},
		"autoreply":   {fn: cmdAutoreply, minLevel: levelModerator},
		"xkcd":        {fn: cmdXKCD, minLevel: levelSubscriber},
		"raffle":      {fn: cmdRaffle, minLevel: levelEveryone},
		"var":         {fn: cmdVar, minLevel: levelModerator},
		"__roundtrip": {fn: cmdRoundtrip, minLevel: levelAdmin},
	}
}

type handlerMap map[string]handlerFunc

func (h handlerMap) run(ctx context.Context, s *session, cmd string, args string) (bool, error) {
	cmd = strings.ToLower(cmd)
	bc, ok := h[cmd]
	if !ok {
		return false, nil
	}

	defer s.UsageContext(cmd)()

	return true, bc.run(ctx, s, cmd, args)
}

type handlerFunc struct {
	fn       func(ctx context.Context, s *session, cmd string, args string) error
	minLevel accessLevel
}

func (h handlerFunc) run(ctx context.Context, s *session, cmd string, args string) error {
	if !s.UserLevel.CanAccess(h.minLevel) {
		return errNotAuthorized
	}

	return h.fn(ctx, s, cmd, args)
}
