package twitch

import (
	"context"
	"encoding/json"

	"golang.org/x/oauth2"
)

// GetIDForToken gets the Twitch user ID for the specified token.
//
// GET https://api.twitch.tv/kraken
func (t *Twitch) GetIDForToken(ctx context.Context, userToken *oauth2.Token) (id int64, newToken *oauth2.Token, err error) {
	cli := t.clientForUser(ctx, true, userToken, setToken(&newToken))

	resp, err := cli.Get(ctx, krakenRoot)
	if err != nil {
		return 0, newToken, err
	}

	defer resp.Body.Close()

	if err := statusToError(resp.StatusCode); err != nil {
		return 0, newToken, err
	}

	out := &struct {
		Token struct {
			UserID IDStr `json:"user_id"`
		} `json:"token"`
	}{}

	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return 0, newToken, ErrServerError
	}

	return out.Token.UserID.AsInt64(), newToken, nil
}
