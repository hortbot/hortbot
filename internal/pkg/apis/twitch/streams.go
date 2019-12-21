package twitch

import (
	"context"
	"strconv"
	"time"

	"github.com/hortbot/hortbot/internal/pkg/jsonx"
)

// Stream represents a Twitch stream as described by the Kraken
// streams endpoint. Some fields are missing (but may be added as
// needed in the future).
type Stream struct {
	ID        IDStr     `json:"_id"`
	Game      string    `json:"game"`
	Viewers   int64     `json:"viewers"`
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

	if err := jsonx.DecodeSingle(resp.Body, &v); err != nil {
		return nil, ErrServerError
	}

	return v.Stream, nil
}
