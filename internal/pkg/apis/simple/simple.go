package simple

import (
	"context"
	"io"
	"io/ioutil"
	"net/http"

	"golang.org/x/net/context/ctxhttp"
)

const (
	plaintextLimit = 512
)

//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6 -generate

//counterfeiter:generate . API

type API interface {
	Plaintext(ctx context.Context, u string) (body string, code int, err error)
}

type Client struct {
	cli *http.Client
}

var _ API = (*Client)(nil)

// New creates a new Urban Dictionary client.
func New(opts ...Option) *Client {
	c := &Client{}

	for _, opt := range opts {
		opt(c)
	}

	return c
}

// Option controls client functionality.
type Option func(*Client)

// HTTPClient sets the Urban client's underlying http.Client.
// If nil (or if this option wasn't used), http.DefaultClient will be used.
func HTTPClient(cli *http.Client) Option {
	return func(c *Client) {
		c.cli = cli
	}
}

func (c *Client) Plaintext(ctx context.Context, u string) (body string, code int, err error) {
	resp, err := ctxhttp.Get(ctx, c.cli, u)
	if err != nil {
		return "", 0, err
	}
	defer resp.Body.Close()

	lr := &io.LimitedReader{
		R: resp.Body,
		N: plaintextLimit,
	}

	b, err := ioutil.ReadAll(lr)
	if err != nil {
		return "", 0, err
	}

	return string(b), resp.StatusCode, nil
}
