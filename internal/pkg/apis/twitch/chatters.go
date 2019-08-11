package twitch

import (
	"context"
	"encoding/json"
	"strings"

	"golang.org/x/net/context/ctxhttp"
)

type Chatters struct {
	Count    int64 `json:"chatter_count"`
	Chatters struct {
		Broadcaster []string `json:"broadcaster"`
		Vips        []string `json:"vips"`
		Moderators  []string `json:"moderators"`
		Staff       []string `json:"staff"`
		Admins      []string `json:"admins"`
		GlobalMods  []string `json:"global_mods"`
		Viewers     []string `json:"viewers"`
	} `json:"chatters"`
}

// GetChatters gets the chatters for a channel.
//
// GET https://tmi.twitch.tv/group/user/<channel>/chatters
func (t *Twitch) GetChatters(ctx context.Context, channel string) (*Chatters, error) {
	url := "https://tmi.twitch.tv/group/user/" + strings.ToLower(channel) + "/chatters"

	resp, err := ctxhttp.Get(ctx, t.cli, url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if err := statusToError(resp.StatusCode); err != nil {
		return nil, err
	}

	c := &Chatters{}

	if err := json.NewDecoder(resp.Body).Decode(c); err != nil {
		return nil, ErrServerError
	}

	return c, nil
}
