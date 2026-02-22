package twitch

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/carlmjohnson/requests"
	"github.com/hortbot/hortbot/internal/pkg/apiclient"
	"github.com/hortbot/hortbot/internal/pkg/httpx"
	"github.com/hortbot/hortbot/internal/pkg/jsonx"
	"github.com/zikaeroh/ctxlog"
	"go.uber.org/zap"
	"golang.org/x/oauth2"
)

type httpClient struct {
	cli     httpx.Client
	ts      oauth2.TokenSource
	headers http.Header
}

func (h *httpClient) finishRequest(ctx context.Context, req *requests.Builder) (*requests.Builder, error) {
	tok, err := h.ts.Token()
	if err != nil {
		if oauthErr, ok := errors.AsType[*oauth2.RetrieveError](err); ok {
			var body struct {
				Error   string `json:"error"`
				Status  int    `json:"status"`
				Message string `json:"message"`
			}

			if err := json.Unmarshal(oauthErr.Body, &body); err != nil {
				return nil, apiclient.NewNonStatusError("twitch", err)
			}

			ctxlog.Info(ctx, "dead oauth token", zap.Any("body", &body))

			if !strings.EqualFold(body.Message, "Invalid refresh token") {
				ctxlog.Warn(ctx, "unknown oauth token error message", zap.String("error_message", body.Message))
			}

			return nil, apiclient.NewNonStatusError("twitch", ErrDeadToken)
		}
		return nil, fmt.Errorf("getting token: %w", err)
	}

	req.Headers(h.headers)
	req.Header("Authorization", tok.Type()+" "+tok.AccessToken)

	req.AddValidator(func(r *http.Response) error {
		reqErr := requests.DefaultValidator(r)
		if reqErr == nil {
			return nil
		}

		if r.StatusCode == http.StatusNotFound {
			// 404 is often just an empty array, so don't collect an error.
			return reqErr //nolint:wrapcheck
		}

		// Status code was an error; try and read the body as JSON
		var body json.RawMessage
		if err := jsonx.DecodeSingle(r.Body, &body); err != nil {
			return reqErr //nolint:wrapcheck
		}

		return fmt.Errorf("%w: %s", reqErr, body)
	})

	return req, nil
}

func (h *httpClient) NewRequest(ctx context.Context, url string) (*requests.Builder, error) {
	return h.finishRequest(ctx, h.cli.NewRequest(url))
}

func (h *httpClient) NewRequestToJSON(ctx context.Context, url string, v any) (*requests.Builder, error) {
	return h.finishRequest(ctx, h.cli.NewRequestToJSON(url, v))
}
