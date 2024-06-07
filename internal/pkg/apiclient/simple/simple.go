// Package simple implements a simple HTTP client for accessing URLs.
package simple

import (
	"context"
	"io"
	"net/http"

	"github.com/hortbot/hortbot/internal/pkg/apiclient"
	"github.com/hortbot/hortbot/internal/pkg/httpx"
)

const (
	plaintextLimit = 512
)

//go:generate go run github.com/matryer/moq -fmt goimports -out simplemocks/mocks.go -pkg simplemocks . API

// API represents the supported API functions. It's defined for fake generation.
type API interface {
	Plaintext(ctx context.Context, u string) (body string, err error)
}

// Client is a simple HTTP client to fetch URLs.
type Client struct {
	cli httpx.Client
}

var _ API = (*Client)(nil)

// New creates a new Urban Dictionary client.
func New(cli *http.Client) *Client {
	return &Client{
		cli: httpx.NewClient(cli, "simple"),
	}
}

// Plaintext gets the specified URL as text.
func (c *Client) Plaintext(ctx context.Context, u string) (body string, err error) {
	var s string

	req := c.cli.NewRequest(u).AddValidator(nil).Handle(func(r *http.Response) error {
		lr := &io.LimitedReader{
			R: r.Body,
			N: plaintextLimit,
		}

		b, err := io.ReadAll(lr)
		if err != nil {
			return err //nolint:wrapcheck
		}

		s = string(b)
		return nil
	})
	if err := req.Fetch(ctx); err != nil {
		return "", apiclient.WrapRequestErr("simple", err, nil)
	}

	return s, nil
}
