package twitch

import (
	"context"
	"encoding/json"
	"strconv"

	"golang.org/x/oauth2"
)

// Channel represents a Twitch channel as described by the Kraken
// channel/channels endpoint. Some fields are missing (but may be added as
// needed in the future).
type Channel struct {
	ID     int64  `json:"_id,string"` // Unbelievably (believably?), the Twitch API reference says this a number, but their server sends a string. Really?
	Game   string `json:"game"`
	Status string `json:"status"`
}

// GetChannelByID gets a channel using the client's token.
//
// GET https://api.twitch.tv/kraken/channels/<id>
func (t *Twitch) GetChannelByID(ctx context.Context, id int64) (c *Channel, err error) {
	cli := t.krakenCli

	url := krakenRoot + "/channels/" + strconv.FormatInt(id, 10)

	resp, err := cli.Get(ctx, url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if err := statusToError(resp.StatusCode); err != nil {
		return nil, err
	}

	c = &Channel{}

	if err := json.NewDecoder(resp.Body).Decode(c); err != nil {
		return nil, ErrServerError
	}

	return c, nil
}

// SetChannelStatus sets the channel's status.
//
// PUT https://api.twitch.tv/kraken/channels/<id>
func (t *Twitch) SetChannelStatus(ctx context.Context, id int64, userToken *oauth2.Token, status string) (newToken *oauth2.Token, err error) {
	if userToken == nil || userToken.AccessToken == "" {
		return nil, ErrNotAuthorized
	}

	cli := t.clientForUser(ctx, true, userToken, setToken(&newToken))

	url := krakenRoot + "/channels/" + strconv.FormatInt(id, 10)

	body := &struct {
		Status string `json:"status"`
	}{
		Status: status,
	}

	resp, err := cli.Put(ctx, url, body)
	if err != nil {
		return newToken, err
	}
	defer resp.Body.Close()

	return newToken, statusToError(resp.StatusCode)
}

// SetChannelGame sets the channel's game. If empty, the game will be unset.
//
// PUT https://api.twitch.tv/kraken/channels/<id>
func (t *Twitch) SetChannelGame(ctx context.Context, id int64, userToken *oauth2.Token, game string) (newToken *oauth2.Token, err error) {
	if userToken == nil || userToken.AccessToken == "" {
		return nil, ErrNotAuthorized
	}

	cli := t.clientForUser(ctx, true, userToken, setToken(&newToken))

	url := krakenRoot + "/channels/" + strconv.FormatInt(id, 10)

	body := &struct {
		Game string `json:"game"`
	}{
		Game: game,
	}

	resp, err := cli.Put(ctx, url, body)
	if err != nil {
		return newToken, err
	}
	defer resp.Body.Close()

	return newToken, statusToError(resp.StatusCode)
}
