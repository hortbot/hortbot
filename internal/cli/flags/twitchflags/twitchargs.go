// Package twitchflags processes Twitch client related flags.
package twitchflags

import "github.com/hortbot/hortbot/internal/pkg/apis/twitch"

type Twitch struct {
	TwitchClientID     string `long:"twitch-client-id" env:"HB_TWITCH_CLIENT_ID" description:"Twitch OAuth client ID" required:"true"`
	TwitchClientSecret string `long:"twitch-client-secret" env:"HB_TWITCH_CLIENT_SECRET" description:"Twitch OAuth client secret" required:"true"`
	TwitchRedirectURL  string `long:"twitch-redirect-url" env:"HB_TWITCH_REDIRECT_URL" description:"Twitch OAuth redirect URL" required:"true"`
}

var DefaultTwitch = Twitch{}

func (args *Twitch) TwitchClient() *twitch.Twitch {
	return twitch.New(args.TwitchClientID, args.TwitchClientSecret, args.TwitchRedirectURL)
}
