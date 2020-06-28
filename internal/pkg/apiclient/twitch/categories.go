package twitch

import (
	"context"
	"net/url"
	"strconv"

	"github.com/hortbot/hortbot/internal/pkg/jsonx"
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

	resp, err := cli.Get(ctx, url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if err := statusToError(resp.StatusCode); err != nil {
		if err == ErrNotFound {
			return []*Category{}, nil
		}
		return nil, err
	}

	body := &struct {
		Data []*Category `json:"data"`
	}{}

	if err := jsonx.DecodeSingle(resp.Body, body); err != nil {
		return nil, ErrServerError
	}

	return body.Data, nil
}

// GetGame queries for a game by name. The name must match exactly.
//
// GET https://api.twitch.tv/helix/games?name=<name>
func (t *Twitch) GetGameByName(ctx context.Context, name string) (*Category, error) {
	return t.getGame(ctx, "name="+url.QueryEscape(name))
}

// GetGame queries for a game by ID.
//
// GET https://api.twitch.tv/helix/games?id=<id>
func (t *Twitch) GetGameByID(ctx context.Context, id int64) (*Category, error) {
	return t.getGame(ctx, "id="+strconv.FormatInt(id, 10))
}

func (t *Twitch) getGame(ctx context.Context, params string) (*Category, error) {
	cli := t.helixCli
	url := helixRoot + "/games?" + params

	resp, err := cli.Get(ctx, url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if err := statusToError(resp.StatusCode); err != nil {
		return nil, err
	}

	body := &struct {
		Data []*Category `json:"data"`
	}{}

	if err := jsonx.DecodeSingle(resp.Body, body); err != nil {
		return nil, ErrServerError
	}

	if len(body.Data) == 0 {
		return nil, ErrNotFound
	}

	return body.Data[0], nil
}
