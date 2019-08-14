package tinyurl

import (
	"context"
	"errors"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"

	"golang.org/x/net/context/ctxhttp"
)

//go:generate gobin -m -run github.com/maxbrunsfeld/counterfeiter/v6 -generate

var ErrServerError = errors.New("tinyurl: server error")

//counterfeiter:generate . API
type API interface {
	Shorten(ctx context.Context, url string) (shortened string, err error)
}

type TinyURL struct {
	cli *http.Client
}

var _ API = (*TinyURL)(nil)

func New(opts ...Option) *TinyURL {
	t := &TinyURL{}

	for _, opt := range opts {
		opt(t)
	}

	return t
}

type Option func(*TinyURL)

// HTTPClient sets the TinyURL client's underlying http.Client.
// If nil (or if this option wasn't used), http.DefaultClient will be used.
func HTTPClient(cli *http.Client) Option {
	return func(s *TinyURL) {
		s.cli = cli
	}
}

func (t *TinyURL) Shorten(ctx context.Context, u string) (shortened string, err error) {
	u = "https://tinyurl.com/api-create.php?url=" + url.QueryEscape(u)

	resp, err := ctxhttp.Get(ctx, t.cli, u)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", ErrServerError
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(body)), nil
}
