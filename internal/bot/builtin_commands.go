package bot

var builtinCommands handlerMap

var reservedCommandNames = map[string]bool{
	"builtin": true,
	"command": true,
	"set":     true,
}

func init() {
	// To prevent initialization loop.
	builtinCommands = newHandlerMap(map[string]handlerFunc{
		"command":         {fn: cmdCommand, minLevel: AccessLevelModerator},
		"coemand":         {fn: cmdCommand, minLevel: AccessLevelModerator},
		"set":             {fn: cmdSettings, minLevel: AccessLevelModerator},
		"setting":         {fn: cmdSettings, minLevel: AccessLevelModerator},
		"owner":           {fn: cmdOwnerModRegularIgnore, minLevel: AccessLevelBroadcaster},
		"mod":             {fn: cmdOwnerModRegularIgnore, minLevel: AccessLevelBroadcaster},
		"regular":         {fn: cmdOwnerModRegularIgnore, minLevel: AccessLevelModerator},
		"ignore":          {fn: cmdOwnerModRegularIgnore, minLevel: AccessLevelModerator},
		"quote":           {fn: cmdQuote, minLevel: AccessLevelSubscriber},
		"clear":           {fn: cmdModClear, minLevel: AccessLevelModerator},
		"filter":          {fn: cmdFilter, minLevel: AccessLevelModerator},
		"permit":          {fn: cmdPermit, minLevel: AccessLevelModerator},
		"allow":           {fn: cmdPermit, minLevel: AccessLevelModerator},
		"leave":           {fn: cmdLeave, minLevel: AccessLevelBroadcaster},
		"part":            {fn: cmdLeave, minLevel: AccessLevelBroadcaster},
		"conch":           {fn: cmdConch, minLevel: AccessLevelSubscriber},
		"helix":           {fn: cmdConch, minLevel: AccessLevelSubscriber},
		"repeat":          {fn: cmdRepeat, minLevel: AccessLevelModerator},
		"schedule":        {fn: cmdSchedule, minLevel: AccessLevelModerator},
		"lastfm":          {fn: cmdLastFM, minLevel: AccessLevelEveryone, skipCooldown: true},
		"songlink":        {fn: cmdSonglink, minLevel: AccessLevelEveryone, skipCooldown: true},
		"music":           {fn: cmdMusic, minLevel: AccessLevelEveryone, skipCooldown: true},
		"autoreply":       {fn: cmdAutoreply, minLevel: AccessLevelModerator},
		"xkcd":            {fn: cmdXKCD, minLevel: AccessLevelSubscriber, skipCooldown: true},
		"raffle":          {fn: cmdRaffle, minLevel: AccessLevelEveryone, skipCooldown: true},
		"var":             {fn: cmdVar, minLevel: AccessLevelModerator},
		"status":          {fn: cmdStatus, minLevel: AccessLevelEveryone},
		"game":            {fn: cmdGame, minLevel: AccessLevelEveryone},
		"viewers":         {fn: cmdViewers, minLevel: AccessLevelEveryone},
		"uptime":          {fn: cmdUptime, minLevel: AccessLevelEveryone},
		"admin":           {fn: cmdAdmin, minLevel: AccessLevelAdmin},
		"islive":          {fn: cmdIsLive, minLevel: AccessLevelModerator},
		"list":            {fn: cmdList, minLevel: AccessLevelModerator},
		"random":          {fn: cmdRandom, minLevel: AccessLevelEveryone, skipCooldown: true},
		"roll":            {fn: cmdRandom, minLevel: AccessLevelEveryone, skipCooldown: true},
		"host":            {fn: cmdHost, minLevel: AccessLevelEveryone},
		"unhost":          {fn: cmdUnhost, minLevel: AccessLevelEveryone},
		"whatshouldiplay": {fn: cmdWhatShouldIPlay, minLevel: AccessLevelBroadcaster},
		"statusgame":      {fn: cmdStatusGame, minLevel: AccessLevelModerator},
		"steamgame":       {fn: cmdSteamGame, minLevel: AccessLevelModerator},
		"google":          {fn: cmdGoogle, minLevel: AccessLevelSubscriber},
		"link":            {fn: cmdLink, minLevel: AccessLevelSubscriber},
		"urban":           {fn: cmdUrban, minLevel: AccessLevelSubscriber, skipCooldown: true},
		"commands":        {fn: cmdCommands, minLevel: AccessLevelSubscriber},
		"coemands":        {fn: cmdCommands, minLevel: AccessLevelSubscriber},
		"quotes":          {fn: cmdQuotes, minLevel: AccessLevelSubscriber},
		"bothelp":         {fn: cmdHelp, minLevel: AccessLevelEveryone},
		"help":            {fn: cmdHelp, minLevel: AccessLevelEveryone},
		"channelid":       {fn: cmdChannelID, minLevel: AccessLevelEveryone},
		"ht":              {fn: cmdHighlight, minLevel: AccessLevelEveryone, skipCooldown: true},
		"highlightthat":   {fn: cmdHighlight, minLevel: AccessLevelEveryone, skipCooldown: true},
		"hltb":            {fn: cmdHLTB, minLevel: AccessLevelSubscriber},
	})

	builtinCommands.isBuiltins = true
}

func isBuiltinName(name string) bool {
	_, ok := builtinCommands.m[name]
	return ok
}
