package bot

import (
	"context"
	"strings"

	"go.opencensus.io/trace"
)

var builtinCommands handlerMap

var reservedCommandNames = map[string]bool{
	"builtin": true,
	"command": true,
	"set":     true,
}

func init() {
	// To prevent initialization loop.
	builtinCommands = newHandlerMap(map[string]handlerFunc{
		"command":         {fn: cmdCommand, minLevel: levelModerator},
		"coemand":         {fn: cmdCommand, minLevel: levelModerator},
		"set":             {fn: cmdSettings, minLevel: levelModerator},
		"setting":         {fn: cmdSettings, minLevel: levelModerator},
		"owner":           {fn: cmdOwnerModRegularIgnore, minLevel: levelBroadcaster},
		"mod":             {fn: cmdOwnerModRegularIgnore, minLevel: levelBroadcaster},
		"regular":         {fn: cmdOwnerModRegularIgnore, minLevel: levelBroadcaster},
		"ignore":          {fn: cmdOwnerModRegularIgnore, minLevel: levelModerator},
		"quote":           {fn: cmdQuote, minLevel: levelSubscriber},
		"clear":           {fn: cmdModClear, minLevel: levelModerator},
		"filter":          {fn: cmdFilter, minLevel: levelModerator},
		"permit":          {fn: cmdPermit, minLevel: levelModerator},
		"allow":           {fn: cmdPermit, minLevel: levelModerator},
		"leave":           {fn: cmdLeave, minLevel: levelBroadcaster},
		"part":            {fn: cmdLeave, minLevel: levelBroadcaster},
		"conch":           {fn: cmdConch, minLevel: levelSubscriber},
		"helix":           {fn: cmdConch, minLevel: levelSubscriber},
		"repeat":          {fn: cmdRepeat, minLevel: levelModerator},
		"schedule":        {fn: cmdSchedule, minLevel: levelModerator},
		"lastfm":          {fn: cmdLastFM, minLevel: levelEveryone, skipCooldown: true},
		"songlink":        {fn: cmdSonglink, minLevel: levelEveryone, skipCooldown: true},
		"music":           {fn: cmdMusic, minLevel: levelEveryone, skipCooldown: true},
		"autoreply":       {fn: cmdAutoreply, minLevel: levelModerator},
		"xkcd":            {fn: cmdXKCD, minLevel: levelSubscriber, skipCooldown: true},
		"raffle":          {fn: cmdRaffle, minLevel: levelEveryone, skipCooldown: true},
		"var":             {fn: cmdVar, minLevel: levelModerator},
		"status":          {fn: cmdStatus, minLevel: levelEveryone},
		"game":            {fn: cmdGame, minLevel: levelEveryone},
		"viewers":         {fn: cmdViewers, minLevel: levelEveryone},
		"uptime":          {fn: cmdUptime, minLevel: levelEveryone},
		"chatters":        {fn: cmdChatters, minLevel: levelEveryone},
		"admin":           {fn: cmdAdmin, minLevel: levelAdmin},
		"islive":          {fn: cmdIsLive, minLevel: levelModerator},
		"ishere":          {fn: cmdIsHere, minLevel: levelModerator},
		"list":            {fn: cmdList, minLevel: levelModerator},
		"random":          {fn: cmdRandom, minLevel: levelEveryone, skipCooldown: true},
		"roll":            {fn: cmdRandom, minLevel: levelEveryone, skipCooldown: true},
		"host":            {fn: cmdHost, minLevel: levelEveryone},
		"unhost":          {fn: cmdUnhost, minLevel: levelEveryone},
		"whatshouldiplay": {fn: cmdWhatShouldIPlay, minLevel: levelBroadcaster},
		"statusgame":      {fn: cmdStatusGame, minLevel: levelModerator},
		"steamgame":       {fn: cmdSteamGame, minLevel: levelModerator},
		"winner":          {fn: cmdWinner, minLevel: levelModerator},
		"google":          {fn: cmdGoogle, minLevel: levelSubscriber},
		"link":            {fn: cmdLink, minLevel: levelSubscriber},
		"followme":        {fn: cmdFollowMe, minLevel: levelBroadcaster},
		"urban":           {fn: cmdUrban, minLevel: levelSubscriber, skipCooldown: true},
		"commands":        {fn: cmdCommands, minLevel: levelSubscriber},
		"coemands":        {fn: cmdCommands, minLevel: levelSubscriber},
		"quotes":          {fn: cmdQuotes, minLevel: levelSubscriber},
		"bothelp":         {fn: cmdHelp, minLevel: levelEveryone},
		"help":            {fn: cmdHelp, minLevel: levelEveryone},
	})
}

func isBuiltinName(name string) bool {
	_, ok := builtinCommands[name]
	return ok
}

type handlerMap map[string]handlerFunc

func verifyHandlerMapEntry(name string, hf handlerFunc) {
	if name == "" {
		panic("empty name")
	}

	if name != strings.ToLower(name) {
		panic("name is not lowercase")
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
	return m
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

	bc, ok := h[cmd]
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

	return true, bc.run(ctx, s, cmd, args)
}

type handlerFunc struct {
	fn           func(ctx context.Context, s *session, cmd string, args string) error
	minLevel     accessLevel
	skipCooldown bool
}

func (h handlerFunc) run(ctx context.Context, s *session, cmd string, args string) error {
	if !s.UserLevel.CanAccess(h.minLevel) {
		return errNotAuthorized
	}

	return h.fn(ctx, s, cmd, args)
}
