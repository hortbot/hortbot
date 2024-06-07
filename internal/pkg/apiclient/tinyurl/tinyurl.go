// Package tinyurl provides a TinyURL client.
package tinyurl

import (
	"context"
	"net/http"
	"strings"

	"github.com/hortbot/hortbot/internal/pkg/apiclient"
	"github.com/hortbot/hortbot/internal/pkg/httpx"
)

//go:generate go run github.com/matryer/moq -fmt goimports -out tinyurlmocks/mocks.go -pkg tinyurlmocks . API

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
func New(cli *http.Client) *TinyURL {
	return &TinyURL{
		cli: httpx.NewClient(cli, "tinyurl"),
	}
}

// Shorten shortens the given URL using TinyURL.
func (t *TinyURL) Shorten(ctx context.Context, u string) (shortened string, err error) {
	var body string

	req := t.cli.NewRequest("https://tinyurl.com/api-create.php").Param("url", u).ToString(&body)
	if err := req.Fetch(ctx); err != nil {
		return "", apiclient.WrapRequestErr("tinyurl", err, nil)
	}

	return strings.TrimSpace(body), nil
}
