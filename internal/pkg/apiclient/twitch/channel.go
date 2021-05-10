package twitch

import (
	"context"
	"strconv"

	"github.com/hortbot/hortbot/internal/pkg/jsonx"
	"golang.org/x/oauth2"
)

// ChannelModerator is a channel's moderator.
type ChannelModerator struct {
	ID   IDStr  `json:"user_id"`
	Name string `json:"user_name"`
}

// GetChannelModerators gets the channel's moderators.
//
// GET https://api.twitch.tv/helix/moderation/moderators
func (t *Twitch) GetChannelModerators(ctx context.Context, id int64, userToken *oauth2.Token) (mods []*ChannelModerator, newToken *oauth2.Token, err error) {
	cursor := ""

	doOne := func() error {
		url := helixRoot + "/moderation/moderators?broadcaster_id=" + strconv.FormatInt(id, 10)
		if cursor != "" {
			url += "&after=" + cursor
		}

		cli := t.helixClientForUser(ctx, userToken, setToken(&newToken))

		resp, err := cli.Get(ctx, url)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		if err := statusToError(resp.StatusCode); err != nil {
			return err
		}

		var v struct {
			Mods       []*ChannelModerator `json:"data"`
			Pagination struct {
				Cursor string `json:"cursor"`
			} `json:"pagination"`
		}

		if err := jsonx.DecodeSingle(resp.Body, &v); err != nil {
			return ErrServerError
		}

		mods = append(mods, v.Mods...)
		cursor = v.Pagination.Cursor

		return nil
	}

	prevLen := 0

	for {
		if err := doOne(); err != nil {
			return nil, newToken, err
		}

		if cursor == "" {
			break
		}

		// Sanity checks.
		if len(mods) == prevLen || len(mods) >= 500 {
			break
		}

		prevLen = len(mods)
	}

	return mods, newToken, nil
}

// ModifyChannel modifies a channel. Either or both of the title and game ID must be provided.
// The title must not be empty. If zero, the game will be unset.
//
// PATCH https://api.twitch.tv/helix/channels
func (t *Twitch) ModifyChannel(ctx context.Context, broadcasterID int64, userToken *oauth2.Token, title *string, gameID *int64) (newToken *oauth2.Token, err error) {
	if title == nil && gameID == nil {
		return nil, ErrBadRequest
	}

	if title != nil && *title == "" {
		return nil, ErrBadRequest
	}

	if userToken == nil || userToken.AccessToken == "" {
		return nil, ErrNotAuthorized
	}

	cli := t.helixClientForUser(ctx, userToken, setToken(&newToken))
	url := helixRoot + "/channels"

	body := &struct {
		BroadcasterID IDStr   `json:"broadcaster_id"`
		Title         *string `json:"title,omitempty"`
		GameID        *IDStr  `json:"game_id,omitempty"`
	}{
		BroadcasterID: IDStr(broadcasterID),
		Title:         title,
		GameID:        (*IDStr)(gameID),
	}

	resp, err := cli.Patch(ctx, url, body)
	if err != nil {
		return newToken, err
	}
	defer resp.Body.Close()

	return newToken, statusToError(resp.StatusCode)
}

// Channel is a channel as exposed by the Helix API.
type Channel struct {
	ID     IDStr  `json:"broadcaster_id"`
	Name   string `json:"broadcaster_name"`
	Game   string `json:"game_name"`
	GameID IDStr  `json:"game_id"`
	Title  string `json:"title"`
}

// GetChannelByID gets a channel using the client's token.
//
// GET https://api.twitch.tv/helix/channels?broadcaster_id<id>
func (t *Twitch) GetChannelByID(ctx context.Context, id int64) (*Channel, error) {
	cli := t.helixCli
	url := helixRoot + "/channels?broadcaster_id=" + strconv.FormatInt(id, 10)

	resp, err := cli.Get(ctx, url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if err := statusToError(resp.StatusCode); err != nil {
		return nil, err
	}

	body := &struct {
		Data []*Channel `json:"data"`
	}{}

	if err := jsonx.DecodeSingle(resp.Body, body); err != nil {
		return nil, ErrServerError
	}

	if len(body.Data) == 0 {
		return nil, ErrNotFound
	}

	return body.Data[0], nil
}
