package twitch

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/carlmjohnson/requests"
	"github.com/hortbot/hortbot/internal/pkg/apiclient"
	"github.com/hortbot/hortbot/internal/pkg/errorsx"
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

func (h *httpClient) finishRequest(ctx context.Context, req *requests.Builder) (*requests.Builder, error) {
	tok, err := h.ts.Token()
	if err != nil {
		if oauthErr, ok := errorsx.As[*oauth2.RetrieveError](err); ok {
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

	// TODO: Use ErrorJson to improve error handling?

	return req, nil
}

func (h *httpClient) NewRequest(ctx context.Context, url string) (*requests.Builder, error) {
	return h.finishRequest(ctx, h.cli.NewRequest(url))
}

func (h *httpClient) NewRequestToJSON(ctx context.Context, url string, v any) (*requests.Builder, error) {
	return h.finishRequest(ctx, h.cli.NewRequestToJSON(url, v))
}
