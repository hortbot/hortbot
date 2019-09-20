package modelsx

import "github.com/hortbot/hortbot/internal/db/models"

const DefaultPrefix = "!"

// NewChannel creates a new Channel with the defaults set.
func NewChannel(userID int64, name, displayName string, botName string) *models.Channel {
	return &models.Channel{
		UserID:                  userID,
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
	}
}
