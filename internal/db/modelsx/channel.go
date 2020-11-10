package modelsx

import "github.com/hortbot/hortbot/internal/db/models"

// DefaultPrefix is the default prefix inserted for new channels. The default
// prefix is not NULL, as changing it based on some config would break users
// who had previously been using the original prefix.
const DefaultPrefix = "!"

// NewChannel creates a new Channel with the defaults set.
func NewChannel(twitchID int64, name, displayName string, botName string) *models.Channel {
	return &models.Channel{
		TwitchID:                twitchID,
		Name:                    name,
		DisplayName:             displayName,
		BotName:                 botName,
		Active:                  true,
		Prefix:                  DefaultPrefix,
		Mode:                    models.AccessLevelEveryone,
		ShouldModerate:          true,
		EnableWarnings:          true,
		SubsMayLink:             true,
		TimeoutDuration:         600,
		RollLevel:               models.AccessLevelSubscriber,
		RollCooldown:            10,
		RollDefault:             20,
		FilterCapsPercentage:    50,
		FilterCapsMinCaps:       6,
		FilterSymbolsPercentage: 50,
		FilterSymbolsMinSymbols: 5,
		FilterMaxLength:         500,
		FilterEmotesMax:         4,
		Tweet:                   "Check out (_CHANNEL_URL_) playing (_GAME_) on @Twitch!",
		FilterExemptLevel:       models.AccessLevelSubscriber,
	}
}
