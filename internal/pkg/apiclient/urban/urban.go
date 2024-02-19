// Package urban provides an Urban Dictionary API client.
package urban

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"strings"

	"github.com/hortbot/hortbot/internal/pkg/httpx"
	"github.com/hortbot/hortbot/internal/pkg/jsonx"
)

// Urban Dictionary API errors.
var (
	ErrNotFound    = errors.New("urban: not found")
	ErrServerError = errors.New("urban: server error")
	ErrUnknown     = errors.New("urban: unknown error")
)

//go:generate go run github.com/matryer/moq -fmt goimports -out urbanmocks/mocks.go -pkg urbanmocks . API

// API represents the supported API functions. It's defined for fake generation.
type API interface {
	Define(ctx context.Context, s string) (string, error)
}

// Urban is an Urban Dictionary API client.
type Urban struct {
	cli httpx.Client
}

var _ API = (*Urban)(nil)

// New creates a new Urban Dictionary client.
func New(cli *http.Client) *Urban {
	return &Urban{
		cli: httpx.NewClient(cli, "urban", false),
	}
}

// Define queries Urban Dictionary for the top definition for a term. The
// returned definition will be stripped of cross-linking square brackets.
func (u *Urban) Define(ctx context.Context, term string) (string, error) {
	ur := "https://api.urbandictionary.com/v0/define?term=" + url.QueryEscape(term)

	resp, err := u.cli.Get(ctx, ur)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if err := statusToError(resp.StatusCode); err != nil {
		return "", err
	}

	body := struct {
		List []struct {
			Definition string `json:"definition"`
		} `json:"list"`
	}{}

	if err := jsonx.DecodeSingle(resp.Body, &body); err != nil {
		return "", ErrServerError
	}

	if len(body.List) == 0 {
		return "", ErrNotFound
	}

	// Urban only uses square brackets for cross linking; they will never appear
	// otherwise.
	def := strings.Map(func(r rune) rune {
		switch r {
		case '[', ']':
			return -1
		default:
			return r
		}
	}, body.List[0].Definition)

	return def, nil
}

func statusToError(code int) error {
	if code >= 200 && code < 300 {
		return nil
	}

	if code == http.StatusNotFound {
		return ErrNotFound
	}

	if code >= 500 {
		return ErrServerError
	}

	return ErrUnknown
}
