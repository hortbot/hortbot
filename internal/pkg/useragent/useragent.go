package useragent


import (
	"github.com/hortbot/hortbot/internal/version"
)


var botUserAgent = "HortBot/" + version.Version()

// Bot returns the bot's user agent string.
func Bot() string {
	return botUserAgent
}
