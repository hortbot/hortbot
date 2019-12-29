package tinyurl_test

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"testing"

	"github.com/hortbot/hortbot/internal/pkg/apis/tinyurl"
	"github.com/jarcoal/httpmock"
	"gotest.tools/v3/assert"
)

func newTransport(t *testing.T) *httpmock.MockTransport {
	t.Helper()
	mt := httpmock.NewMockTransport()
	mt.RegisterNoResponder(httpmock.NewNotFoundResponder(t.Fatal))
	return mt
}

func TestShorten(t *testing.T) {
	const longURL = "https://github.com/hortbot/hortbot"
	query := "url=" + url.QueryEscape(longURL)

	ctx := context.Background()

	t.Run("Good", func(t *testing.T) {
		const shortURL = "https://tinyurl.com/2tx"

		mt := newTransport(t)
		mt.RegisterResponderWithQuery("GET", "https://tinyurl.com/api-create.php", query, httpmock.NewStringResponder(200, shortURL))

		tu := tinyurl.New(tinyurl.HTTPClient(&http.Client{Transport: mt}))

		short, err := tu.Shorten(ctx, longURL)
		assert.NilError(t, err)
		assert.Equal(t, short, shortURL)
	})

	t.Run("Request error", func(t *testing.T) {
		testErr := errors.New("testing error")

		mt := newTransport(t)
		mt.RegisterResponderWithQuery("GET", "https://tinyurl.com/api-create.php", query, httpmock.NewErrorResponder(testErr))

		tu := tinyurl.New(tinyurl.HTTPClient(&http.Client{Transport: mt}))

		_, err := tu.Shorten(ctx, longURL)
		assert.ErrorContains(t, err, testErr.Error())
	})

	t.Run("Good", func(t *testing.T) {
		mt := newTransport(t)
		mt.RegisterResponderWithQuery("GET", "https://tinyurl.com/api-create.php", query, httpmock.NewStringResponder(400, ""))

		tu := tinyurl.New(tinyurl.HTTPClient(&http.Client{Transport: mt}))

		_, err := tu.Shorten(ctx, longURL)
		assert.Equal(t, err, tinyurl.ErrServerError)
	})

	t.Run("ReadAll error", func(t *testing.T) {
		response := httpmock.NewStringResponse(200, "") //nolint:bodyclose
		response.Body = (*badBody)(nil)

		mt := newTransport(t)
		mt.RegisterResponderWithQuery("GET", "https://tinyurl.com/api-create.php", query, httpmock.ResponderFromResponse(response))

		tu := tinyurl.New(tinyurl.HTTPClient(&http.Client{Transport: mt}))

		_, err := tu.Shorten(ctx, longURL)
		assert.Equal(t, err, tinyurl.ErrServerError)
	})
}

type badBody struct{}

func (*badBody) Read(p []byte) (n int, err error) {
	return 0, errors.New("bad body")
}

func (*badBody) Close() error {
	return nil
}
