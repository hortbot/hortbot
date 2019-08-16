package urban_test

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"testing"

	"github.com/hortbot/hortbot/internal/pkg/apis/urban"
	"github.com/jarcoal/httpmock"
	"gotest.tools/assert"
)

func newTransport(t *testing.T) *httpmock.MockTransport {
	t.Helper()
	mt := httpmock.NewMockTransport()
	mt.RegisterNoResponder(httpmock.NewNotFoundResponder(t.Fatal))
	return mt
}

func TestDefine(t *testing.T) {
	const (
		apiURL = "https://api.urbandictionary.com/v0/define"
		phrase = "something"
	)

	query := "term=" + url.QueryEscape(phrase)

	ctx := context.Background()

	t.Run("Good", func(t *testing.T) {
		mt := newTransport(t)
		mt.RegisterResponderWithQuery("GET", apiURL, query, httpmock.NewStringResponder(200, `{"list": [{"definition": "This is [some] definition. [Wow]."}, {"definition": "This is a [second] definition. [Wow]."}]}`))

		ti := urban.New(urban.HTTPClient(&http.Client{Transport: mt}))

		def, err := ti.Define(ctx, phrase)
		assert.NilError(t, err)
		assert.Equal(t, def, "This is some definition. Wow.")
	})

	t.Run("Request error", func(t *testing.T) {
		testErr := errors.New("testing error")

		mt := newTransport(t)
		mt.RegisterResponderWithQuery("GET", apiURL, query, httpmock.NewErrorResponder(testErr))

		ti := urban.New(urban.HTTPClient(&http.Client{Transport: mt}))

		_, err := ti.Define(ctx, phrase)
		assert.ErrorContains(t, err, testErr.Error())
	})

	t.Run("Not found", func(t *testing.T) {
		mt := newTransport(t)
		mt.RegisterResponderWithQuery("GET", apiURL, query, httpmock.NewStringResponder(404, ""))

		ti := urban.New(urban.HTTPClient(&http.Client{Transport: mt}))

		_, err := ti.Define(ctx, phrase)
		assert.Equal(t, err, urban.ErrNotFound)
	})

	t.Run("Empty", func(t *testing.T) {
		mt := newTransport(t)
		mt.RegisterResponderWithQuery("GET", apiURL, query, httpmock.NewStringResponder(200, "{}"))

		ti := urban.New(urban.HTTPClient(&http.Client{Transport: mt}))

		_, err := ti.Define(ctx, phrase)
		assert.Equal(t, err, urban.ErrNotFound)
	})

	t.Run("Server error", func(t *testing.T) {
		mt := newTransport(t)
		mt.RegisterResponderWithQuery("GET", apiURL, query, httpmock.NewStringResponder(500, ""))

		ti := urban.New(urban.HTTPClient(&http.Client{Transport: mt}))

		_, err := ti.Define(ctx, phrase)
		assert.Equal(t, err, urban.ErrServerError)
	})

	t.Run("Unknown error", func(t *testing.T) {
		mt := newTransport(t)
		mt.RegisterResponderWithQuery("GET", apiURL, query, httpmock.NewStringResponder(418, ""))

		ti := urban.New(urban.HTTPClient(&http.Client{Transport: mt}))

		_, err := ti.Define(ctx, phrase)
		assert.Equal(t, err, urban.ErrUnknown)
	})

	t.Run("Bad JSON", func(t *testing.T) {
		mt := newTransport(t)
		mt.RegisterResponderWithQuery("GET", apiURL, query, httpmock.NewStringResponder(200, "}"))

		ti := urban.New(urban.HTTPClient(&http.Client{Transport: mt}))

		_, err := ti.Define(ctx, phrase)
		assert.Equal(t, err, urban.ErrServerError)
	})
}
