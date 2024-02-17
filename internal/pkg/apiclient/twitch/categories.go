package twitch

import (
	"context"
	"net/url"
	"strconv"
)

type Category struct {
	ID   IDStr  `json:"id"`
	Name string `json:"name"`
}

// SearchCategories searches for categories that match the specified query.
//
// GET https://api.twitch.tv/helix/search/categories?query=<query>
func (t *Twitch) SearchCategories(ctx context.Context, query string) ([]*Category, error) {
	cli := t.helixCli
	url := helixRoot + "/search/categories?query=" + url.QueryEscape(query)
	return fetchList[*Category](ctx, cli, url)
}

// GetGameByName queries for a game by name. The name must match exactly.
//
// GET https://api.twitch.tv/helix/games?name=<name>
func (t *Twitch) GetGameByName(ctx context.Context, name string) (*Category, error) {
	return t.getGame(ctx, "name="+url.QueryEscape(name))
}

// GetGameByID queries for a game by ID.
//
// GET https://api.twitch.tv/helix/games?id=<id>
func (t *Twitch) GetGameByID(ctx context.Context, id int64) (*Category, error) {
	return t.getGame(ctx, "id="+strconv.FormatInt(id, 10))
}

func (t *Twitch) getGame(ctx context.Context, params string) (*Category, error) {
	cli := t.helixCli
	url := helixRoot + "/games?" + params
	return fetchFirstFromList[*Category](ctx, cli, url)
}
