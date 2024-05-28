package twitch

import (
	"context"
	"strconv"
	"time"

	"github.com/hortbot/hortbot/internal/pkg/apiclient/twitch/idstr"
)

type Stream struct {
	ID          idstr.IDStr `json:"id"`
	GameID      idstr.IDStr `json:"game_id"`
	Title       string      `json:"title"`
	StartedAt   time.Time   `json:"started_at"`
	ViewerCount int         `json:"viewer_count"`
	UserID      idstr.IDStr `json:"user_id"`
}

// GetStreamByUserID gets the current stream by user ID.
//
// GET https://api.twitch.tv/helix/streams?user_id=<id>
func (t *Twitch) GetStreamByUserID(ctx context.Context, id int64) (*Stream, error) {
	req, err := t.helixCli.NewRequest(ctx, helixRoot+"/streams")
	if err != nil {
		return nil, err
	}
	req.Param("user_id", strconv.FormatInt(id, 10))
	return fetchFirstFromList[*Stream](ctx, req)
}

// GetStreamByUsername gets the current stream by username.
//
// GET https://api.twitch.tv/helix/streams?user_login=<username>
func (t *Twitch) GetStreamByUsername(ctx context.Context, username string) (*Stream, error) {
	req, err := t.helixCli.NewRequest(ctx, helixRoot+"/streams")
	if err != nil {
		return nil, err
	}
	req.Param("user_login", username)
	return fetchFirstFromList[*Stream](ctx, req)
}
