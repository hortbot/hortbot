// Package xkcd provides an XKCD API client.
package xkcd

import (
	"context"
	"net/http"
	"strconv"

	"github.com/hortbot/hortbot/internal/pkg/apiclient"
	"github.com/hortbot/hortbot/internal/pkg/httpx"
)

// Comic is an XKCD comic.
type Comic struct {
	Title string `json:"safe_title"`
	Img   string `json:"img"`
	Alt   string `json:"alt"`
}

//go:generate go run github.com/matryer/moq -fmt goimports -out xkcdmocks/mocks.go -pkg xkcdmocks . API

// API represents the supported API functions. It's defined for fake generation.
type API interface {
	GetComic(ctx context.Context, id int) (*Comic, error)
}

// XKCD is an XKCD API client.
type XKCD struct {
	cli httpx.Client
}

var _ API = &XKCD{}

// New creates a new XKCD API client.
func New(cli *http.Client) *XKCD {
	return &XKCD{
		cli: httpx.NewClient(cli, "xkcd"),
	}
}

// GetComic fetches the specified XKCD comic.
func (x *XKCD) GetComic(ctx context.Context, id int) (*Comic, error) {
	url := "https://xkcd.com/" + strconv.Itoa(id) + "/info.0.json"

	c := &Comic{}

	if err := x.cli.NewRequestToJSON(url, c).Fetch(ctx); err != nil {
		return nil, apiclient.WrapRequestErr("xkcd", err, nil)
	}

	return c, nil
}
