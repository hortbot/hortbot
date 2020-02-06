// Package twitchflags processes Twitch client related flags.
package twitchflags

import (
	"net/http"

	"github.com/hortbot/hortbot/internal/pkg/apis/twitch"
)

// Twitch contains Twitch API client flags.
type Twitch struct {
	ClientID     string `long:"twitch-client-id" env:"HB_TWITCH_CLIENT_ID" description:"Twitch OAuth client ID" required:"true"`
	ClientSecret string `long:"twitch-client-secret" env:"HB_TWITCH_CLIENT_SECRET" description:"Twitch OAuth client secret" required:"true"`
	RedirectURL  string `long:"twitch-redirect-url" env:"HB_TWITCH_REDIRECT_URL" description:"Twitch OAuth redirect URL" required:"true"`
}

// Default contains the default flags. Make a copy of this, do not reuse.
var Default = Twitch{}

// Client returns a new Twitch API client.
func (args *Twitch) Client(httpClient *http.Client) *twitch.Twitch {
	return twitch.New(args.ClientID, args.ClientSecret, args.RedirectURL, twitch.HTTPClient(httpClient))
}
