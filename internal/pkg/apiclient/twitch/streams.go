package twitch

import (
	"context"
	"strconv"
	"time"

	"github.com/hortbot/hortbot/internal/pkg/jsonx"
)

type Stream struct {
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
func (t *Twitch) GetStreamByUserID(ctx context.Context, id int64) (*Stream, error) {
	return t.getStream(ctx, "user_id="+strconv.FormatInt(id, 10))
}

// GetStreamByUsername gets the current stream by username.
//
// GET https://api.twitch.tv/helix/streams?user_login=<username>
func (t *Twitch) GetStreamByUsername(ctx context.Context, username string) (*Stream, error) {
	return t.getStream(ctx, "user_login="+username)
}

func (t *Twitch) getStream(ctx context.Context, query string) (*Stream, error) {
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
		Data []*Stream `json:"data"`
	}{}

	if err := jsonx.DecodeSingle(resp.Body, body); err != nil {
		return nil, ErrServerError
	}

	if len(body.Data) == 0 {
		return nil, ErrNotFound
	}

	return body.Data[0], nil
}
