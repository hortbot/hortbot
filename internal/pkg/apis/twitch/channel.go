package twitch

import (
	"context"
	"strconv"

	"github.com/hortbot/hortbot/internal/pkg/jsonx"
	"golang.org/x/oauth2"
)

// Channel represents a Twitch channel as described by the Kraken
// channel/channels endpoint. Some fields are missing (but may be added as
// needed in the future).
type Channel struct {
	ID          IDStr  `json:"_id"`
	Name        string `json:"name"`
	DisplayName string `json:"display_name"`
	Game        string `json:"game"`
	Status      string `json:"status"`
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

	if err := jsonx.DecodeSingle(resp.Body, c); err != nil {
		return nil, ErrServerError
	}

	return c, nil
}

// SetChannelStatus sets the channel's status.
// The status the API returns in response to this request will be returned, and
// can be checked to verify that the status was updated.
//
// PUT https://api.twitch.tv/kraken/channels/<id>
func (t *Twitch) SetChannelStatus(ctx context.Context, id int64, userToken *oauth2.Token, status string) (newStatus string, newToken *oauth2.Token, err error) {
	if userToken == nil || userToken.AccessToken == "" {
		return "", nil, ErrNotAuthorized
	}

	cli := t.clientForUser(ctx, true, userToken, setToken(&newToken))

	url := krakenRoot + "/channels/" + strconv.FormatInt(id, 10)

	body := &struct {
		Channel struct {
			Status string `json:"status"`
		} `json:"channel"`
	}{}
	body.Channel.Status = status

	resp, err := cli.Put(ctx, url, body)
	if err != nil {
		return "", newToken, err
	}
	defer resp.Body.Close()

	c := &Channel{}

	if err := jsonx.DecodeSingle(resp.Body, c); err != nil {
		return "", newToken, ErrServerError
	}

	// TODO: Return the entire channel?
	return c.Status, newToken, statusToError(resp.StatusCode)
}

// SetChannelGame sets the channel's game. If empty, the game will be unset.
// The game the API returns in response to this request will be returned, and
// can be checked to verify that the status was updated.
//
// PUT https://api.twitch.tv/kraken/channels/<id>
func (t *Twitch) SetChannelGame(ctx context.Context, id int64, userToken *oauth2.Token, game string) (newGame string, newToken *oauth2.Token, err error) {
	if userToken == nil || userToken.AccessToken == "" {
		return "", nil, ErrNotAuthorized
	}

	cli := t.clientForUser(ctx, true, userToken, setToken(&newToken))

	url := krakenRoot + "/channels/" + strconv.FormatInt(id, 10)

	body := &struct {
		Channel struct {
			Game string `json:"game"`
		} `json:"channel"`
	}{}
	body.Channel.Game = game

	resp, err := cli.Put(ctx, url, body)
	if err != nil {
		return "", newToken, err
	}
	defer resp.Body.Close()

	c := &Channel{}

	if err := jsonx.DecodeSingle(resp.Body, c); err != nil {
		return "", newToken, ErrServerError
	}

	// TODO: Return the entire channel?
	return c.Game, newToken, statusToError(resp.StatusCode)
}
