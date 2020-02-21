package simple_test

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/hortbot/hortbot/internal/pkg/apis"
	"github.com/hortbot/hortbot/internal/pkg/apis/simple"
	"github.com/hortbot/hortbot/internal/pkg/httpmockx"
	"github.com/jarcoal/httpmock"
	"gotest.tools/v3/assert"
)

func TestDefine(t *testing.T) {
	ctx := context.Background()
	const apiURL = "https://example.com/something"

	t.Run("Good", func(t *testing.T) {
		mt := httpmockx.NewMockTransport(t)
		mt.RegisterResponder("GET", apiURL, httpmock.NewStringResponder(200, "This is some text."))

		sc := simple.New(simple.HTTPClient(&http.Client{Transport: mt}))

		body, err := sc.Plaintext(ctx, apiURL)
		assert.NilError(t, err)
		assert.Equal(t, body, "This is some text.")
	})

	t.Run("Request error", func(t *testing.T) {
		testErr := errors.New("testing error")

		mt := httpmockx.NewMockTransport(t)
		mt.RegisterResponder("GET", apiURL, httpmock.NewErrorResponder(testErr))

		sc := simple.New(simple.HTTPClient(&http.Client{Transport: mt}))

		_, err := sc.Plaintext(ctx, apiURL)
		assert.ErrorContains(t, err, testErr.Error())
	})

	t.Run("Not found", func(t *testing.T) {
		mt := httpmockx.NewMockTransport(t)
		mt.RegisterResponder("GET", apiURL, httpmock.NewStringResponder(404, "not found"))

		sc := simple.New(simple.HTTPClient(&http.Client{Transport: mt}))

		body, err := sc.Plaintext(ctx, apiURL)
		assert.Equal(t, body, "not found")
		assert.DeepEqual(t, err, &apis.Error{API: "simple", StatusCode: 404})
	})

	t.Run("Server error", func(t *testing.T) {
		mt := httpmockx.NewMockTransport(t)
		mt.RegisterResponder("GET", apiURL, httpmock.NewStringResponder(500, "error!"))

		sc := simple.New(simple.HTTPClient(&http.Client{Transport: mt}))

		body, err := sc.Plaintext(ctx, apiURL)
		assert.Equal(t, body, "error!")
		assert.DeepEqual(t, err, &apis.Error{API: "simple", StatusCode: 500})
	})

	t.Run("Limit", func(t *testing.T) {
		text := "this is a test"

		for i := 0; i < 10; i++ {
			text += text
		}

		mt := httpmockx.NewMockTransport(t)
		mt.RegisterResponder("GET", apiURL, httpmock.NewStringResponder(201, text))

		sc := simple.New(simple.HTTPClient(&http.Client{Transport: mt}))

		body, err := sc.Plaintext(ctx, apiURL)
		assert.NilError(t, err)
		assert.Equal(t, len(body), 512)
	})

	t.Run("ReadAll error", func(t *testing.T) {
		response := httpmock.NewStringResponse(200, "") //nolint:bodyclose
		response.Body = (*badBody)(nil)

		mt := httpmockx.NewMockTransport(t)
		mt.RegisterResponder("GET", apiURL, httpmock.ResponderFromResponse(response))

		sc := simple.New(simple.HTTPClient(&http.Client{Transport: mt}))

		_, err := sc.Plaintext(ctx, apiURL)
		assert.Equal(t, err, errBadBody)
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
