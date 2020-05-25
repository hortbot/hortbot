package twitch

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/hortbot/hortbot/internal/pkg/ctxlog"
	"go.uber.org/zap"
	"golang.org/x/net/context/ctxhttp"
	"golang.org/x/oauth2"
)

type httpClient struct {
	cli     *http.Client
	ts      oauth2.TokenSource
	headers http.Header
}

func (h *httpClient) newRequest(ctx context.Context, method string, url string, body io.Reader) (*http.Request, error) {
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}

	tok, err := h.ts.Token()
	if err != nil {
		var oauthErr *oauth2.RetrieveError
		if errors.As(err, &oauthErr) {
			ctxlog.Error(ctx, "oauth retrieve error occurred", zap.ByteString("body", oauthErr.Body))
		}
		return nil, err
	}

	if h.headers == nil {
		req.Header = make(http.Header)
	} else {
		req.Header = h.headers.Clone()
	}

	tok.SetAuthHeader(req)

	return req, nil
}

func (h *httpClient) Get(ctx context.Context, url string) (*http.Response, error) {
	req, err := h.newRequest(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	return h.do(ctx, req)
}

func (h *httpClient) Put(ctx context.Context, url string, v interface{}) (*http.Response, error) {
	var buf bytes.Buffer

	if err := json.NewEncoder(&buf).Encode(v); err != nil {
		return nil, err
	}

	req, err := h.newRequest(ctx, "PUT", url, &buf)
	if err != nil {
		return nil, err
	}

	return h.do(ctx, req)
}

func (h *httpClient) do(ctx context.Context, req *http.Request) (*http.Response, error) {
	// x, _ := httputil.DumpRequestOut(req, true)
	// log.Printf("%s", x)

	resp, err := ctxhttp.Do(ctx, h.cli, req)
	if err != nil {
		var oauthErr *oauth2.RetrieveError
		if errors.As(err, &oauthErr) {
			ctxlog.Error(ctx, "oauth retrieve error occurred", zap.ByteString("body", oauthErr.Body))
		}
		return nil, err
	}

	// y, _ := httputil.DumpResponse(resp, true)
	// log.Printf("%s", y)

	return resp, nil
}
