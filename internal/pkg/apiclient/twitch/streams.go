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
	Channel   *Channel  `json:"channel"`
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

type HelixStream struct {
	ID          IDStr     `json:"id"`
	GameID      IDStr     `json:"game_id"`
	Title       string    `json:"title"`
	StartedAt   time.Time `json:"started_at"`
	ViewerCount int       `json:"viewer_count"`
	UserID      IDStr     `json:"user_id"`
}

// GetStreamByUserID gets the current stream by user ID.
//
// GET https://api.twitch.tv/helix/streams?user_id=<id>
func (t *Twitch) GetStreamByUserID(ctx context.Context, id int64) (*HelixStream, error) {
	return t.getStream(ctx, "user_id="+strconv.FormatInt(id, 10))
}

// GetStreamByUserID gets the current stream by username.
//
// GET https://api.twitch.tv/helix/streams?user_login=<username>
func (t *Twitch) GetStreamByUsername(ctx context.Context, username string) (*HelixStream, error) {
	return t.getStream(ctx, "user_login="+username)
}

func (t *Twitch) getStream(ctx context.Context, query string) (*HelixStream, error) {
	cli := t.helixCli
	url := helixRoot + "/streams?" + query

	resp, err := cli.Get(ctx, url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if err := statusToError(resp.StatusCode); err != nil {
		return nil, err
	}

	body := &struct {
		Data []*HelixStream `json:"data"`
	}{}

	if err := jsonx.DecodeSingle(resp.Body, body); err != nil {
		return nil, ErrServerError
	}

	if len(body.Data) == 0 {
		return nil, ErrNotFound
	}

	return body.Data[0], nil
}
