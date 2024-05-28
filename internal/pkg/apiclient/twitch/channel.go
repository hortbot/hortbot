package twitch

import (
	"context"
	"net/http"
	"strconv"

	"github.com/hortbot/hortbot/internal/pkg/apiclient"
	"github.com/hortbot/hortbot/internal/pkg/apiclient/twitch/idstr"
	"golang.org/x/oauth2"
)

// ChannelModerator is a channel's moderator.
type ChannelModerator struct {
	ID   idstr.IDStr `json:"user_id"`
	Name string      `json:"user_name"`
}

// GetChannelModerators gets the channel's moderators.
//
// GET https://api.twitch.tv/helix/moderation/moderators
func (t *Twitch) GetChannelModerators(ctx context.Context, id int64, userToken *oauth2.Token) (mods []*ChannelModerator, newToken *oauth2.Token, err error) {
	cli := t.clientForUser(ctx, userToken, setToken(&newToken))
	req, err := cli.NewRequest(ctx, helixRoot+"/moderation/moderators")
	if err != nil {
		return nil, newToken, err
	}
	req.Param("broadcaster_id", strconv.FormatInt(id, 10))
	mods, err = paginate[*ChannelModerator](ctx, req, 100, 500)
	return mods, newToken, err
}

// ModifyChannel modifies a channel. Either or both of the title and game ID must be provided.
// The title must not be empty. If zero, the game will be unset.
//
// PATCH https://api.twitch.tv/helix/channels
func (t *Twitch) ModifyChannel(ctx context.Context, broadcasterID int64, userToken *oauth2.Token, title *string, gameID *int64) (newToken *oauth2.Token, err error) {
	if title == nil && gameID == nil {
		return nil, apiclient.NewStatusError("twitch", http.StatusBadRequest)
	}

	if title != nil && *title == "" {
		return nil, apiclient.NewStatusError("twitch", http.StatusBadRequest)
	}

	if userToken == nil || userToken.AccessToken == "" {
		return nil, apiclient.NewStatusError("twitch", http.StatusUnauthorized)
	}

	cli := t.clientForUser(ctx, userToken, setToken(&newToken))
	url := helixRoot + "/channels"

	body := &struct {
		BroadcasterID idstr.IDStr  `json:"broadcaster_id"`
		Title         *string      `json:"title,omitempty"`
		GameID        *idstr.IDStr `json:"game_id,omitempty"`
	}{
		BroadcasterID: idstr.IDStr(broadcasterID),
		Title:         title,
		GameID:        (*idstr.IDStr)(gameID),
	}

	req, err := cli.NewRequest(ctx, url)
	if err != nil {
		return newToken, err
	}

	if err := req.BodyJSON(body).Patch().Fetch(ctx); err != nil {
		return newToken, apiclient.WrapRequestErr("twitch", err)
	}

	return newToken, nil
}

// Channel is a channel as exposed by the Helix API.
type Channel struct {
	ID     idstr.IDStr `json:"broadcaster_id"`
	Name   string      `json:"broadcaster_name"`
	Game   string      `json:"game_name"`
	GameID idstr.IDStr `json:"game_id"`
	Title  string      `json:"title"`
}

// GetChannelByID gets a channel using the client's token.
//
// GET https://api.twitch.tv/helix/channels?broadcaster_id<id>
func (t *Twitch) GetChannelByID(ctx context.Context, id int64) (*Channel, error) {
	req, err := t.helixCli.NewRequest(ctx, helixRoot+"/channels")
	if err != nil {
		return nil, err
	}
	req.Param("broadcaster_id", strconv.FormatInt(id, 10))
	return fetchFirstFromList[*Channel](ctx, req)
}
