package urban

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strings"

	"golang.org/x/net/context/ctxhttp"
)

//go:generate gobin -m -run github.com/maxbrunsfeld/counterfeiter/v6 -generate

var (
	ErrNotFound    = errors.New("urban: not found")
	ErrServerError = errors.New("urban: server error")
	ErrUnknown     = errors.New("urban: unknown error")
)

//counterfeiter:generate . API
type API interface{}

type Urban struct {
	cli *http.Client
}

var _ API = (*Urban)(nil)

func New(opts ...Option) *Urban {
	t := &Urban{}

	for _, opt := range opts {
		opt(t)
	}

	return t
}

type Option func(*Urban)

// HTTPClient sets the Urban client's underlying http.Client.
// If nil (or if this option wasn't used), http.DefaultClient will be used.
func HTTPClient(cli *http.Client) Option {
	return func(s *Urban) {
		s.cli = cli
	}
}

func (u *Urban) Define(ctx context.Context, s string) (string, error) {
	ur := "https://api.urbandictionary.com/v0/define?term=" + url.QueryEscape(s)

	resp, err := ctxhttp.Get(ctx, u.cli, ur)
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

	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
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
