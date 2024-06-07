// Package httpx implements HTTP helper types and functions.
package httpx

import (
	"context"
	"net/http"
	"time"

	"github.com/carlmjohnson/requests"
	"github.com/hortbot/hortbot/internal/pkg/jsonx"
	"github.com/hortbot/hortbot/internal/pkg/useragent"
	"golang.org/x/oauth2"
)

// Client is an HTTP client with context helpers.
//
// This type is similar to ctxhttp's functions, but does not copy requests unnecessarily.
type Client struct {
	client    *http.Client
	asBrowser bool
}

func NewClient(cli *http.Client, name string, opts ...Option) Client {
	if cli == nil {
		panic("nil http.Client")
	}

	client := *cli

	c := Client{
		client: &client,
	}

	for _, opt := range opts {
		opt(&c)
	}

	transport := client.Transport
	if transport == nil {
		transport = http.DefaultTransport
	}
	client.Transport = &wrappedTransport{
		inner:     transport,
		asBrowser: c.asBrowser,
		name:      name,
	}

	return c
}

type Option func(*Client)

func WithBrowserUserAgent() Option {
	return func(c *Client) {
		c.asBrowser = true
	}
}

type wrappedTransport struct {
	inner     http.RoundTripper
	asBrowser bool
	name      string
}

func (w *wrappedTransport) RoundTrip(req *http.Request) (resp *http.Response, err error) {
	start := time.Now()
	defer func() {
		end := time.Now()

		code := 0
		if resp != nil {
			code = resp.StatusCode
		}

		labels := makeLabels(w.name, req.Method, code)

		metricRequests.With(labels).Inc()
		if err != nil {
			metricErrors.With(labels).Inc()
		} else {
			metricRequestDuration.With(labels).Observe(end.Sub(start).Seconds())
		}
	}()

	const userAgentHeader = "User-Agent"

	if _, ok := req.Header[userAgentHeader]; !ok {
		if w.asBrowser {
			req.Header.Set(userAgentHeader, useragent.Browser())
		} else {
			req.Header.Set(userAgentHeader, useragent.Bot())
		}
	}

	return w.inner.RoundTrip(req) //nolint:wrapcheck
}

func (c *Client) NewRequest(url string) *requests.Builder {
	return requests.URL(url).Client(c.client)
}

func (c *Client) NewRequestToJSON(url string, v any) *requests.Builder {
	return requests.URL(url).Client(c.client).Handle(ToJSON(v)) //nolint:bodyclose
}

// ToJSON is like [requests.ToJSON] but verifies that only a single value is decoded.
func ToJSON(v any) requests.ResponseHandler {
	return func(r *http.Response) error {
		return jsonx.DecodeSingle(r.Body, v)
	}
}

// AsOAuth2Client registers the client as the client to use in the OAuth2 library.
// The context returned by this function should be used when calling OAuth2 related functions.
func (c *Client) AsOAuth2Client(ctx context.Context) context.Context {
	return context.WithValue(ctx, oauth2.HTTPClient, c.client)
}
