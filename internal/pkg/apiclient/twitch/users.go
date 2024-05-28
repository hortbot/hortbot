package twitch

import (
	"context"
	"math"
	"strconv"

	"github.com/hortbot/hortbot/internal/pkg/apiclient/twitch/idstr"
	"golang.org/x/oauth2"
)

// User is a Twitch user.
type User struct {
	ID          idstr.IDStr `json:"id"`
	Name        string      `json:"login"`
	DisplayName string      `json:"display_name,omitempty"`
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
	cli := t.clientForUser(ctx, userToken, setToken(&newToken))
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
	req, err := cli.NewRequest(ctx, u)
	if err != nil {
		return nil, err
	}
	if username != "" {
		req.Param("login", username)
	} else if id != 0 {
		req.Param("id", strconv.FormatInt(id, 10))
	}
	return fetchFirstFromList[*User](ctx, req)
}

type ModeratedChannel struct {
	ID    idstr.IDStr `json:"broadcaster_id"`
	Login string      `json:"broadcaster_login"`
	Name  string      `json:"broadcaster_name"`
}

// GetModeratedChannels gets the channels the user moderates.
//
// GET https://api.twitch.tv/helix/moderation/channels
func (t *Twitch) GetModeratedChannels(ctx context.Context, modID int64, modToken *oauth2.Token) (channels []*ModeratedChannel, newToken *oauth2.Token, err error) {
	cli := t.clientForUser(ctx, modToken, setToken(&newToken))
	u := helixRoot + "/moderation/channels"
	req, err := cli.NewRequest(ctx, u)
	if err != nil {
		return nil, newToken, err
	}
	req.Param("user_id", strconv.FormatInt(modID, 10))
	channels, err = paginate[*ModeratedChannel](ctx, req, 100, math.MaxInt)
	return channels, newToken, err
}
