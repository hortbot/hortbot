package twitch

import (
	"context"
	"strconv"
	"time"
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
	return fetchFirstFromList[*Stream](ctx, cli, url)
}
