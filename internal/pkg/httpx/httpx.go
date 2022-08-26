// Package httpx implements HTTP helper types and functions.
package httpx

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/hortbot/hortbot/internal/pkg/useragent"
	"golang.org/x/oauth2"
)

// Client is an HTTP client with context helpers.
//
// This type is similar to ctxhttp's functions, but does not copy requests unnecessarily.
type Client struct {
	// Client is the HTTP client that will be used. If nil, http.DefaultClient will be used.
	Client *http.Client

	// AsBrowser instructs the client to perform the request like a browser.
	AsBrowser bool

	// Name is the name of the client, for metrics.
	Name string
}

func (c *Client) client() *http.Client {
	if c.Client != nil {
		return c.Client
	}
	return http.DefaultClient
}

func (c *Client) Do(req *http.Request) (resp *http.Response, err error) {
	start := time.Now()
	defer func() {
		end := time.Now()

		code := 0
		if resp != nil {
			code = resp.StatusCode
		}

		labels := makeLabels(c.Name, req.Method, code)

		metricRequests.With(labels).Inc()
		if err != nil {
			metricErrors.With(labels).Inc()
		} else {
			metricRequestDuration.With(labels).Observe(end.Sub(start).Seconds())
		}
	}()

	const userAgentHeader = "User-Agent"

	if _, ok := req.Header[userAgentHeader]; !ok {
		if c.AsBrowser {
			req.Header.Set(userAgentHeader, useragent.Browser())
		} else {
			req.Header.Set(userAgentHeader, useragent.Bot())
		}
	}

	resp, err = c.client().Do(req)
	// If we got an error, and the context has been canceled,
	// the context's error is probably more useful.
	if err != nil {
		ctx := req.Context()
		select {
		case <-ctx.Done():
			err = ctx.Err()
		default:
		}
	}
	return resp, err
}

func (c *Client) DoWithContext(ctx context.Context, req *http.Request) (*http.Response, error) {
	return c.Do(req.WithContext(ctx))
}

func (c *Client) Get(ctx context.Context, url string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	return c.Do(req)
}

func (c *Client) Post(ctx context.Context, url string, bodyType string, body io.Reader, extraHeaders http.Header) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", bodyType)
	for k, v := range extraHeaders {
		req.Header[k] = v
	}
	return c.Do(req)
}

func (c *Client) PostForm(ctx context.Context, url string, data url.Values, extraHeaders http.Header) (*http.Response, error) {
	return c.Post(ctx, url, "application/x-www-form-urlencoded", strings.NewReader(data.Encode()), extraHeaders)
}

// AsOAuth2Client registers the client as the client to use in the OAuth2 library.
// The context returned by this function should be used when calling OAuth2 related functions.
func (c *Client) AsOAuth2Client(ctx context.Context) context.Context {
	return context.WithValue(ctx, oauth2.HTTPClient, c.client())
}
