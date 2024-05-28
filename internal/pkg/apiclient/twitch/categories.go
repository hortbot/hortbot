package twitch

import (
	"context"
	"strconv"

	"github.com/hortbot/hortbot/internal/pkg/apiclient/twitch/idstr"
)

type Category struct {
	ID   idstr.IDStr `json:"id"`
	Name string      `json:"name"`
}

// SearchCategories searches for categories that match the specified query.
//
// GET https://api.twitch.tv/helix/search/categories?query=<query>
func (t *Twitch) SearchCategories(ctx context.Context, query string) ([]*Category, error) {
	req, err := t.helixCli.NewRequest(ctx, helixRoot+"/search/categories")
	if err != nil {
		return nil, err
	}
	req.Param("query", query)
	return fetchList[*Category](ctx, req)
}

// GetGameByName queries for a game by name. The name must match exactly.
//
// GET https://api.twitch.tv/helix/games?name=<name>
func (t *Twitch) GetGameByName(ctx context.Context, name string) (*Category, error) {
	req, err := t.helixCli.NewRequest(ctx, helixRoot+"/games")
	if err != nil {
		return nil, err
	}
	req.Param("name", name)
	return fetchFirstFromList[*Category](ctx, req)
}

// GetGameByID queries for a game by ID.
//
// GET https://api.twitch.tv/helix/games?id=<id>
func (t *Twitch) GetGameByID(ctx context.Context, id int64) (*Category, error) {
	req, err := t.helixCli.NewRequest(ctx, helixRoot+"/games")
	if err != nil {
		return nil, err
	}
	req.Param("id", strconv.FormatInt(id, 10))
	return fetchFirstFromList[*Category](ctx, req)
}
