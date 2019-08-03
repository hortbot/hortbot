package twitch

import (
	"context"
	"encoding/json"
	"net/url"

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

	body := &struct {
		Token struct {
			UserID IDStr `json:"user_id"`
		} `json:"token"`
	}{}

	if err := json.NewDecoder(resp.Body).Decode(body); err != nil {
		return 0, newToken, ErrServerError
	}

	return body.Token.UserID.AsInt64(), newToken, nil
}

// GetIDForUsername gets the Twitch user ID for the specified username.
//
// GET https://api.twitch.tv/kraken/users?login=<username>
func (t *Twitch) GetIDForUsername(ctx context.Context, username string) (int64, error) {
	cli := t.krakenCli

	resp, err := cli.Get(ctx, krakenRoot+"/users?login="+url.QueryEscape(username))
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if err := statusToError(resp.StatusCode); err != nil {
		return 0, err
	}

	body := &struct {
		Users []struct {
			ID int64 `json:"_id"`
		}
	}{}

	if err := json.NewDecoder(resp.Body).Decode(body); err != nil {
		return 0, ErrServerError
	}

	users := body.Users
	if len(users) == 0 {
		return 0, ErrNotFound
	}

	return users[0].ID, nil
}
