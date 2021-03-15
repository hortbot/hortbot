package twitch

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/hortbot/hortbot/internal/pkg/httpx"
	"github.com/zikaeroh/ctxlog"
	"go.uber.org/zap"
	"golang.org/x/oauth2"
)

type httpClient struct {
	cli     httpx.Client
	ts      oauth2.TokenSource
	headers http.Header
}

func (h *httpClient) newRequest(ctx context.Context, method string, url string, body io.Reader) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, err
	}

	tok, err := h.ts.Token()
	if err != nil {
		var oauthErr *oauth2.RetrieveError
		if errors.As(err, &oauthErr) {
			var body struct {
				Error   string `json:"error"`
				Status  int    `json:"status"`
				Message string `json:"message"`
			}

			if err := json.Unmarshal(oauthErr.Body, &body); err != nil {
				return nil, ErrServerError
			}

			ctxlog.Info(ctx, "dead oauth token", zap.Any("body", &body))

			if !strings.EqualFold(body.Message, "Invalid refresh token") {
				ctxlog.Warn(ctx, "unknown oauth token error message", zap.String("error_message", body.Message))
			}

			return nil, ErrDeadToken
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

	return h.do(req)
}

func (h *httpClient) makeJSONRequest(ctx context.Context, method string, url string, v interface{}) (*http.Response, error) {
	var body io.Reader

	if v != nil {
		var buf bytes.Buffer

		if err := json.NewEncoder(&buf).Encode(v); err != nil {
			return nil, err
		}

		body = &buf
	}

	req, err := h.newRequest(ctx, method, url, body)
	if err != nil {
		return nil, err
	}

	if v != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	return h.do(req)
}

func (h *httpClient) Put(ctx context.Context, url string, v interface{}) (*http.Response, error) {
	return h.makeJSONRequest(ctx, "PUT", url, v)
}

func (h *httpClient) Post(ctx context.Context, url string, v interface{}) (*http.Response, error) {
	return h.makeJSONRequest(ctx, "POST", url, v)
}

func (h *httpClient) Patch(ctx context.Context, url string, v interface{}) (*http.Response, error) {
	return h.makeJSONRequest(ctx, "PATCH", url, v)
}

func (h *httpClient) do(req *http.Request) (*http.Response, error) {
	// x, _ := httputil.DumpRequestOut(req, true)
	// log.Printf("%s", x)

	resp, err := h.cli.Do(req)
	if err != nil {
		return nil, err
	}

	// y, _ := httputil.DumpResponse(resp, true)
	// log.Printf("%s", y)

	return resp, nil
}
