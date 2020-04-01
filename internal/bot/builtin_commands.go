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
		"channelid":       {fn: cmdChannelID, minLevel: levelEveryone},
		"ht":              {fn: cmdHighlight, minLevel: levelEveryone, skipCooldown: true},
		"highlightthat":   {fn: cmdHighlight, minLevel: levelEveryone, skipCooldown: true},
		"hltb":            {fn: cmdHLTB, minLevel: levelSubscriber},
	})
}

func isBuiltinName(name string) bool {
	_, ok := builtinCommands[name]
	return ok
}
