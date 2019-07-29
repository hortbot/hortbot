package twitch

import (
	"context"
	"encoding/json"
	"strconv"
	"time"
)

// Stream represents a Twitch stream as described by the Kraken
// streams endpoint. Some fields are missing (but may be added as
// needed in the future).
type Stream struct {
	ID        int64     `json:"_id"`
	Game      string    `json:"game"`
	Viewers   int       `json:"viewers"`
	CreatedAt time.Time `json:"created_at"`
}

// GetCurrentStream gets a channel's current stream. If no stream is active,
// nil is returned.
//
// GET https://api.twitch.tv/kraken/streams/<id>
func (t *Twitch) GetCurrentStream(ctx context.Context, id int64) (s *Stream, err error) {
	cli := t.krakenCli

	url := krakenRoot + "/streams/" + strconv.FormatInt(id, 10)

	resp, err := cli.Get(ctx, url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if err := statusToError(resp.StatusCode); err != nil {
		return nil, err
	}

	var v struct {
		Stream *Stream `json:"stream"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&v); err != nil {
		return nil, ErrServerError
	}

	return v.Stream, nil
}
