package twitch

import (
	"context"
	"encoding/json"
	"strings"

	"golang.org/x/net/context/ctxhttp"
)

// GetChatters gets the number of users connected to the channel's chat.
//
// GET https://tmi.twitch.tv/group/user/<channel>/chatters
func (t *Twitch) GetChatters(ctx context.Context, channel string) (int64, error) {
	url := "https://tmi.twitch.tv/group/user/" + strings.ToLower(channel) + "/chatters"

	resp, err := ctxhttp.Get(ctx, t.cli, url)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if err := statusToError(resp.StatusCode); err != nil {
		return 0, err
	}

	body := &struct {
		ChatterCount int64 `json:"chatter_count"`
	}{}

	if err := json.NewDecoder(resp.Body).Decode(body); err != nil {
		return 0, ErrServerError
	}

	return body.ChatterCount, nil
}
