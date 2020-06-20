package twitch

import (
	"context"
	"net/url"
	"strconv"

	"github.com/hortbot/hortbot/internal/pkg/jsonx"
	"golang.org/x/oauth2"
)

// User is a Twitch user.
type User struct {
	ID          IDStr  `json:"id"`
	Name        string `json:"login"`
	DisplayName string `json:"display_name,omitempty"`
}

// DispName returns the display name for the user if provided, otherwise the username.
func (u User) DispName() string {
	if u.DisplayName != "" {
		return u.DisplayName
	}
	return u.Name
}

// GetUserForToken gets the Twitch user for the specified token.
//
// GET https://api.twitch.tv/helix/users
func (t *Twitch) GetUserForToken(ctx context.Context, userToken *oauth2.Token) (user *User, newToken *oauth2.Token, err error) {
	cli := t.helixClientForUser(ctx, userToken, setToken(&newToken))
	user, err = getUser(ctx, cli, "")
	return user, newToken, err
}

// GetUserForUsername gets the Twitch user for the specified username.
//
// GET https://api.twitch.tv/helix/users?login=<username>
func (t *Twitch) GetUserForUsername(ctx context.Context, username string) (*User, error) {
	return getUser(ctx, t.helixCli, username)
}

func getUser(ctx context.Context, cli *httpClient, username string) (*User, error) {
	u := helixRoot + "/users"
	if username != "" {
		u += "?login=" + url.QueryEscape(username)
	}

	resp, err := cli.Get(ctx, u)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if err := statusToError(resp.StatusCode); err != nil {
		return nil, err
	}

	body := &struct {
		Data []*User `json:"data"`
	}{}

	if err := jsonx.DecodeSingle(resp.Body, body); err != nil {
		return nil, ErrServerError
	}

	users := body.Data
	if len(users) == 0 {
		return nil, ErrNotFound
	}

	return users[0], nil
}

// FollowChannel makes one channel follow another. This requires the
// user_follows_edit scope on the provided token.
//
// PUT https://api.twitch.tv/kraken/users/<id>/follows/channels/<toFollow>
func (t *Twitch) FollowChannel(ctx context.Context, id int64, userToken *oauth2.Token, toFollow int64) (newToken *oauth2.Token, err error) {
	if userToken == nil || userToken.AccessToken == "" {
		return nil, ErrNotAuthorized
	}

	cli := t.krakenClientForUser(ctx, userToken, setToken(&newToken))

	url := krakenRoot + "/users/" + strconv.FormatInt(id, 10) + "/follows/channels/" + strconv.FormatInt(toFollow, 10)

	resp, err := cli.Put(ctx, url, nil)
	if err != nil {
		return newToken, err
	}
	defer resp.Body.Close()

	return newToken, statusToError(resp.StatusCode)
}
