package twitch

import (
	"context"
	"net/url"

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
