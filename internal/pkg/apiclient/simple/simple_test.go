package simple_test

import (
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/hortbot/hortbot/internal/pkg/apiclient/simple"
	"github.com/hortbot/hortbot/internal/pkg/httpmockx"
	"github.com/jarcoal/httpmock"
	"gotest.tools/v3/assert"
)

func TestPlaintext(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	const apiURL = "https://example.com/something"

	t.Run("Good", func(t *testing.T) {
		t.Parallel()
		mt := httpmockx.NewMockTransport(t)
		mt.RegisterResponder("GET", apiURL, httpmock.NewStringResponder(200, "This is some text."))

		sc := simple.New(&http.Client{Transport: mt})

		body, err := sc.Plaintext(ctx, apiURL)
		assert.NilError(t, err)
		assert.Equal(t, body, "This is some text.")
	})

	t.Run("Request error", func(t *testing.T) {
		t.Parallel()
		testErr := errors.New("testing error")

		mt := httpmockx.NewMockTransport(t)
		mt.RegisterResponder("GET", apiURL, httpmock.NewErrorResponder(testErr))

		sc := simple.New(&http.Client{Transport: mt})

		_, err := sc.Plaintext(ctx, apiURL)
		assert.ErrorContains(t, err, testErr.Error())
	})

	t.Run("Not found", func(t *testing.T) {
		t.Parallel()
		mt := httpmockx.NewMockTransport(t)
		mt.RegisterResponder("GET", apiURL, httpmock.NewStringResponder(404, "not found"))

		sc := simple.New(&http.Client{Transport: mt})

		body, err := sc.Plaintext(ctx, apiURL)
		assert.Equal(t, body, "not found")
		assert.NilError(t, err)
	})

	t.Run("Server error", func(t *testing.T) {
		t.Parallel()
		mt := httpmockx.NewMockTransport(t)
		mt.RegisterResponder("GET", apiURL, httpmock.NewStringResponder(500, "error!"))

		sc := simple.New(&http.Client{Transport: mt})

		body, err := sc.Plaintext(ctx, apiURL)
		assert.Equal(t, body, "error!")
		assert.NilError(t, err)
	})

	t.Run("Limit", func(t *testing.T) {
		t.Parallel()
		text := strings.Repeat("x", 513)

		mt := httpmockx.NewMockTransport(t)
		mt.RegisterResponder("GET", apiURL, httpmock.NewStringResponder(201, text))

		sc := simple.New(&http.Client{Transport: mt})

		body, err := sc.Plaintext(ctx, apiURL)
		assert.NilError(t, err)
		assert.Equal(t, len(body), 512)
	})

	t.Run("ReadAll error", func(t *testing.T) {
		t.Parallel()
		response := httpmock.NewStringResponse(200, "") //nolint:bodyclose
		response.Body = (*badBody)(nil)

		mt := httpmockx.NewMockTransport(t)
		mt.RegisterResponder("GET", apiURL, httpmock.ResponderFromResponse(response))

		sc := simple.New(&http.Client{Transport: mt})

		_, err := sc.Plaintext(ctx, apiURL)
		assert.ErrorIs(t, err, errBadBody)
	})
}

var errBadBody = errors.New("bad body")

type badBody struct{}

func (*badBody) Read(p []byte) (n int, err error) {
	return 0, errBadBody
}

func (*badBody) Close() error {
	return nil
}
