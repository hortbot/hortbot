package twitch

import (
	"net/url"
	"strings"
)

// ExpectedUserScopes is the default user scopes as they would appear in a URL.
var ExpectedUserScopes = func() string {
	var b strings.Builder
	for i, scope := range userScopes {
		if i != 0 {
			b.WriteByte('+')
		}

		b.WriteString(url.QueryEscape(scope))
	}

	return b.String()
}()

var GetChannelByID = (*Twitch).getChannelByID
