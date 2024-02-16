// Package tinyurl provides a TinyURL client.
package tinyurl

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/hortbot/hortbot/internal/pkg/httpx"
)

//go:generate go run github.com/matryer/moq -fmt goimports -out tinyurlmocks/mocks.go -pkg tinyurlmocks . API

// ErrServerError is returned when a shortening request is unsuccessful.
var ErrServerError = errors.New("tinyurl: server error")

// API represents the supported API functions. It's defined for fake generation.
type API interface {
	Shorten(ctx context.Context, url string) (shortened string, err error)
}

// TinyURL is a TinyURL API client.
type TinyURL struct {
	cli httpx.Client
}

var _ API = (*TinyURL)(nil)

// New creates a new TinyURL API client.
func New(opts ...Option) *TinyURL {
	t := &TinyURL{
		cli: httpx.Client{
			Name: "tinyurl",
		},
	}

	for _, opt := range opts {
		opt(t)
	}

	return t
}

// Option controls client functionality.
type Option func(*TinyURL)

// HTTPClient sets the TinyURL client's underlying http.Client.
// If nil (or if this option wasn't used), http.DefaultClient will be used.
func HTTPClient(cli *http.Client) Option {
	return func(s *TinyURL) {
		s.cli.Client = cli
	}
}

// Shorten shortens the given URL using TinyURL.
func (t *TinyURL) Shorten(ctx context.Context, u string) (shortened string, err error) {
	u = "https://tinyurl.com/api-create.php?url=" + url.QueryEscape(u)

	resp, err := t.cli.Get(ctx, u)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", ErrServerError
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", ErrServerError
	}

	return strings.TrimSpace(string(body)), nil
}
