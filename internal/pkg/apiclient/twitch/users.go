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

// GetUserByToken gets the Twitch user for the specified token.
//
// GET https://api.twitch.tv/helix/users
func (t *Twitch) GetUserByToken(ctx context.Context, userToken *oauth2.Token) (user *User, newToken *oauth2.Token, err error) {
	cli := t.helixClientForUser(ctx, userToken, setToken(&newToken))
	user, err = getUser(ctx, cli, "", 0)
	return user, newToken, err
}

// GetUserByUsername gets the Twitch user for the specified username.
//
// GET https://api.twitch.tv/helix/users?login=<username>
func (t *Twitch) GetUserByUsername(ctx context.Context, username string) (*User, error) {
	return getUser(ctx, t.helixCli, username, 0)
}

// GetUserByID gets the Twitch user for the specified UD.
//
// GET https://api.twitch.tv/helix/users?id=<id>
func (t *Twitch) GetUserByID(ctx context.Context, id int64) (*User, error) {
	return getUser(ctx, t.helixCli, "", id)
}

func getUser(ctx context.Context, cli *httpClient, username string, id int64) (*User, error) {
	u := helixRoot + "/users"
	if username != "" {
		u += "?login=" + url.QueryEscape(username)
	} else if id != 0 {
		u += "?id=" + strconv.FormatInt(id, 10)
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
