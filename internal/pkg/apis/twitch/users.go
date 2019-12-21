package twitch

import (
	"context"
	"net/url"
	"strconv"

	"github.com/hortbot/hortbot/internal/pkg/jsonx"
	"golang.org/x/oauth2"
)

type User struct {
	ID          int64  `json:"user_id"`
	Name        string `json:"user_name"`
	DisplayName string `json:"display_name,omitempty"`
}

func (u User) DispName() string {
	if u.DisplayName != "" {
		return u.DisplayName
	}
	return u.Name
}

// GetUserForToken gets the Twitch user for the specified token.
//
// GET https://api.twitch.tv/kraken
func (t *Twitch) GetUserForToken(ctx context.Context, userToken *oauth2.Token) (user *User, newToken *oauth2.Token, err error) {
	cli := t.clientForUser(ctx, true, userToken, setToken(&newToken))

	resp, err := cli.Get(ctx, krakenRoot)
	if err != nil {
		return nil, newToken, err
	}
	defer resp.Body.Close()

	if err := statusToError(resp.StatusCode); err != nil {
		return nil, newToken, err
	}

	body := &struct {
		Token struct {
			ID   IDStr  `json:"user_id"`
			Name string `json:"user_name"`
		} `json:"token"`
	}{}

	if err := jsonx.DecodeSingle(resp.Body, body); err != nil {
		return nil, newToken, ErrServerError
	}

	return &User{
		ID:   body.Token.ID.AsInt64(),
		Name: body.Token.Name,
	}, newToken, nil
}

// GetUserForUsername gets the Twitch user for the specified username.
//
// GET https://api.twitch.tv/kraken/users?login=<username>
func (t *Twitch) GetUserForUsername(ctx context.Context, username string) (*User, error) {
	cli := t.krakenCli

	resp, err := cli.Get(ctx, krakenRoot+"/users?login="+url.QueryEscape(username))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if err := statusToError(resp.StatusCode); err != nil {
		return nil, err
	}

	body := &struct {
		Users []struct {
			ID          IDStr  `json:"_id"`
			Name        string `json:"name"`
			DisplayName string `json:"display_name"`
		}
	}{}

	if err := jsonx.DecodeSingle(resp.Body, body); err != nil {
		return nil, ErrServerError
	}

	users := body.Users
	if len(users) == 0 {
		return nil, ErrNotFound
	}

	return &User{
		ID:          users[0].ID.AsInt64(),
		Name:        users[0].Name,
		DisplayName: users[0].DisplayName,
	}, nil
}

// FollowChannel makes one channel follow another. This requires the
// user_follows_edit scope on the provided token.
//
// PUT https://api.twitch.tv/kraken/users/<id>/follows/channels/<toFollow>
func (t *Twitch) FollowChannel(ctx context.Context, id int64, userToken *oauth2.Token, toFollow int64) (newToken *oauth2.Token, err error) {
	if userToken == nil || userToken.AccessToken == "" {
		return nil, ErrNotAuthorized
	}

	cli := t.clientForUser(ctx, true, userToken, setToken(&newToken))

	url := krakenRoot + "/users/" + strconv.FormatInt(id, 10) + "/follows/channels/" + strconv.FormatInt(toFollow, 10)

	resp, err := cli.Put(ctx, url, nil)
	if err != nil {
		return newToken, err
	}
	defer resp.Body.Close()

	return newToken, statusToError(resp.StatusCode)
}
