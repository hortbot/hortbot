// Package urban provides an Urban Dictionary API client.
package urban

import (
	"context"
	"net/http"
	"strings"

	"github.com/hortbot/hortbot/internal/pkg/apiclient"
	"github.com/hortbot/hortbot/internal/pkg/httpx"
)

//go:generate go tool github.com/matryer/moq -fmt goimports -out urbanmocks/mocks.go -pkg urbanmocks . API

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
		cli: httpx.NewClient(cli, "urban"),
	}
}

// Define queries Urban Dictionary for the top definition for a term. The
// returned definition will be stripped of cross-linking square brackets.
func (u *Urban) Define(ctx context.Context, term string) (string, error) {
	var body struct {
		List []struct {
			Definition string `json:"definition"`
		} `json:"list"`
	}

	req := u.cli.NewRequestToJSON("https://api.urbandictionary.com/v0/define", &body).
		Param("term", term)

	if err := req.Fetch(ctx); err != nil {
		return "", apiclient.WrapRequestErr("urban", err, nil)
	}

	if len(body.List) == 0 {
		return "", apiclient.NewStatusError("urban", 404)
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
