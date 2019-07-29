package twitch

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"

	"golang.org/x/net/context/ctxhttp"
	"golang.org/x/oauth2"
)

type httpClient struct {
	cli     *http.Client
	ts      oauth2.TokenSource
	headers http.Header
}

func (h *httpClient) newRequest(method string, url string, body io.Reader) (*http.Request, error) {
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}

	req.Header = cloneHeader(h.headers)

	tok, err := h.ts.Token()
	if err != nil {
		return nil, err
	}
	tok.SetAuthHeader(req)

	return req, nil
}

func (h *httpClient) Get(ctx context.Context, url string) (*http.Response, error) {
	req, err := h.newRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	return ctxhttp.Do(ctx, h.cli, req)
}

func (h *httpClient) Put(ctx context.Context, url string, v interface{}) (*http.Response, error) {
	var buf bytes.Buffer

	if err := json.NewEncoder(&buf).Encode(v); err != nil {
		return nil, err
	}

	req, err := h.newRequest("PUT", url, &buf)
	if err != nil {
		return nil, err
	}

	return ctxhttp.Do(ctx, h.cli, req)
}
