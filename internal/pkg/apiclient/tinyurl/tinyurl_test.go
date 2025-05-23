package tinyurl_test

import (
	"errors"
	"net/http"
	"net/url"
	"testing"

	"github.com/hortbot/hortbot/internal/pkg/apiclient/tinyurl"
	"github.com/hortbot/hortbot/internal/pkg/httpmockx"
	"github.com/jarcoal/httpmock"
	"gotest.tools/v3/assert"
)

func TestShorten(t *testing.T) {
	t.Parallel()
	const longURL = "https://github.com/hortbot/hortbot"
	query := "url=" + url.QueryEscape(longURL)

	ctx := t.Context()

	t.Run("Good", func(t *testing.T) {
		t.Parallel()
		const shortURL = "https://tinyurl.com/2tx"

		mt := httpmockx.NewMockTransport(t)
		mt.RegisterResponderWithQuery("GET", "https://tinyurl.com/api-create.php", query, httpmock.NewStringResponder(200, shortURL))

		tu := tinyurl.New(&http.Client{Transport: mt})

		short, err := tu.Shorten(ctx, longURL)
		assert.NilError(t, err)
		assert.Equal(t, short, shortURL)
	})

	t.Run("Request error", func(t *testing.T) {
		t.Parallel()
		testErr := errors.New("testing error")

		mt := httpmockx.NewMockTransport(t)
		mt.RegisterResponderWithQuery("GET", "https://tinyurl.com/api-create.php", query, httpmock.NewErrorResponder(testErr))

		tu := tinyurl.New(&http.Client{Transport: mt})

		_, err := tu.Shorten(ctx, longURL)
		assert.ErrorContains(t, err, testErr.Error())
	})

	t.Run("Good", func(t *testing.T) {
		t.Parallel()
		mt := httpmockx.NewMockTransport(t)
		mt.RegisterResponderWithQuery("GET", "https://tinyurl.com/api-create.php", query, httpmock.NewStringResponder(400, ""))

		tu := tinyurl.New(&http.Client{Transport: mt})

		_, err := tu.Shorten(ctx, longURL)
		assert.Error(t, err, "tinyurl: ErrValidator: response error for https://tinyurl.com/api-create.php?url=https%3A%2F%2Fgithub.com%2Fhortbot%2Fhortbot: unexpected status: 400")
	})

	t.Run("ReadAll error", func(t *testing.T) {
		t.Parallel()
		response := httpmock.NewStringResponse(200, "") //nolint:bodyclose
		response.Body = (*badBody)(nil)

		mt := httpmockx.NewMockTransport(t)
		mt.RegisterResponderWithQuery("GET", "https://tinyurl.com/api-create.php", query, httpmock.ResponderFromResponse(response))

		tu := tinyurl.New(&http.Client{Transport: mt})

		_, err := tu.Shorten(ctx, longURL)
		assert.Error(t, err, "tinyurl: ErrHandler: bad body")
	})
}

type badBody struct{}

func (*badBody) Read(p []byte) (n int, err error) {
	return 0, errors.New("bad body")
}

func (*badBody) Close() error {
	return nil
}
