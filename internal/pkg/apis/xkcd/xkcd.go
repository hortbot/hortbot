package xkcd

import (
	"context"
	"errors"
	"net/http"
	"strconv"

	"github.com/hortbot/hortbot/internal/pkg/jsonx"
	"golang.org/x/net/context/ctxhttp"
)

var ErrNotFound = errors.New("xkcd: not found")

type Comic struct {
	Title string `json:"safe_title"`
	Img   string `json:"img"`
	Alt   string `json:"alt"`
}

//go:generate gobin -m -run github.com/maxbrunsfeld/counterfeiter/v6 -generate

//counterfeiter:generate . API
type API interface {
	GetComic(ctx context.Context, id int) (*Comic, error)
}

type XKCD struct {
	cli *http.Client
}

var _ API = &XKCD{}

type Option func(*XKCD)

func New(opts ...Option) *XKCD {
	x := &XKCD{}
	for _, opt := range opts {
		opt(x)
	}
	return x
}

// HTTPClient sets the XKCD client's underlying http.Client.
// If nil (or if this option wasn't used), http.DefaultClient will be used.
func HTTPClient(cli *http.Client) Option {
	return func(s *XKCD) {
		s.cli = cli
	}
}

func (x *XKCD) GetComic(ctx context.Context, id int) (*Comic, error) {
	url := "https://xkcd.com/" + strconv.Itoa(id) + "/info.0.json"

	resp, err := ctxhttp.Get(ctx, x.cli, url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return nil, ErrNotFound
	}

	c := &Comic{}

	if err := jsonx.DecodeSingle(resp.Body, c); err != nil {
		return nil, err
	}

	return c, nil
}
