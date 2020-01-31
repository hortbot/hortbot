// Package twitchflags processes Twitch client related flags.
package twitchflags

import (
	"net/http"

	"github.com/hortbot/hortbot/internal/pkg/apis/twitch"
)

type Twitch struct {
	ClientID     string `long:"twitch-client-id" env:"HB_TWITCH_CLIENT_ID" description:"Twitch OAuth client ID" required:"true"`
	ClientSecret string `long:"twitch-client-secret" env:"HB_TWITCH_CLIENT_SECRET" description:"Twitch OAuth client secret" required:"true"`
	RedirectURL  string `long:"twitch-redirect-url" env:"HB_TWITCH_REDIRECT_URL" description:"Twitch OAuth redirect URL" required:"true"`
}

var DefaultTwitch = Twitch{}

func (args *Twitch) Client(httpClient *http.Client) *twitch.Twitch {
	return twitch.New(args.ClientID, args.ClientSecret, args.RedirectURL, twitch.HTTPClient(httpClient))
}
