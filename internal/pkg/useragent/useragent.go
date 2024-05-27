package useragent

// https://github.com/intoli/user-agents

import (
	"github.com/hortbot/hortbot/internal/version"
)

//go:generate go run generate.go

var botUserAgent = "HortBot/" + version.Version()

// Bot returns the bot's user agent string.
func Bot() string {
	return botUserAgent
}

// Browser returns a random browser user agent.
func Browser() string {
	return chooser.Pick()
}
