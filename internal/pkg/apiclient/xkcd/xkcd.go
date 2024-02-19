// Package xkcd provides an XKCD API client.
package xkcd

import (
	"context"
	"errors"
	"net/http"
	"strconv"

	"github.com/hortbot/hortbot/internal/pkg/httpx"
	"github.com/hortbot/hortbot/internal/pkg/jsonx"
)

// ErrNotFound is returned when the requested comic cannot be found.
var ErrNotFound = errors.New("xkcd: not found")

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
		cli: httpx.NewClient(cli, "xkcd", false),
	}
}

// GetComic fetches the specified XKCD comic.
func (x *XKCD) GetComic(ctx context.Context, id int) (*Comic, error) {
	url := "https://xkcd.com/" + strconv.Itoa(id) + "/info.0.json"

	resp, err := x.cli.Get(ctx, url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= http.StatusBadRequest {
		return nil, ErrNotFound
	}

	c := &Comic{}

	if err := jsonx.DecodeSingle(resp.Body, c); err != nil {
		return nil, err
	}

	return c, nil
}
