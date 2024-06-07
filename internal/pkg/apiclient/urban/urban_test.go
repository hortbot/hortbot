package urban_test

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"testing"

	"github.com/hortbot/hortbot/internal/pkg/apiclient/urban"
	"github.com/hortbot/hortbot/internal/pkg/httpmockx"
	"github.com/jarcoal/httpmock"
	"gotest.tools/v3/assert"
)

func TestDefine(t *testing.T) {
	t.Parallel()
	const (
		apiURL = "https://api.urbandictionary.com/v0/define"
		phrase = "something"
	)

	query := "term=" + url.QueryEscape(phrase)

	ctx := context.Background()

	t.Run("Good", func(t *testing.T) {
		t.Parallel()
		mt := httpmockx.NewMockTransport(t)
		mt.RegisterResponderWithQuery("GET", apiURL, query, httpmock.NewStringResponder(200, `{"list": [{"definition": "This is [some] definition. [Wow]."}, {"definition": "This is a [second] definition. [Wow]."}]}`))

		ti := urban.New(&http.Client{Transport: mt})

		def, err := ti.Define(ctx, phrase)
		assert.NilError(t, err)
		assert.Equal(t, def, "This is some definition. Wow.")
	})

	t.Run("Request error", func(t *testing.T) {
		t.Parallel()
		testErr := errors.New("testing error")

		mt := httpmockx.NewMockTransport(t)
		mt.RegisterResponderWithQuery("GET", apiURL, query, httpmock.NewErrorResponder(testErr))

		ti := urban.New(&http.Client{Transport: mt})

		_, err := ti.Define(ctx, phrase)
		assert.ErrorContains(t, err, testErr.Error())
	})

	t.Run("Not found", func(t *testing.T) {
		t.Parallel()
		mt := httpmockx.NewMockTransport(t)
		mt.RegisterResponderWithQuery("GET", apiURL, query, httpmock.NewStringResponder(404, ""))

		ti := urban.New(&http.Client{Transport: mt})

		_, err := ti.Define(ctx, phrase)
		assert.Error(t, err, "urban: ErrValidator: response error for https://api.urbandictionary.com/v0/define?term=something: unexpected status: 404")
	})

	t.Run("Empty", func(t *testing.T) {
		t.Parallel()
		mt := httpmockx.NewMockTransport(t)
		mt.RegisterResponderWithQuery("GET", apiURL, query, httpmock.NewStringResponder(200, "{}"))

		ti := urban.New(&http.Client{Transport: mt})

		_, err := ti.Define(ctx, phrase)
		assert.Error(t, err, "urban: unexpected status: 404")
	})

	t.Run("Server error", func(t *testing.T) {
		t.Parallel()
		mt := httpmockx.NewMockTransport(t)
		mt.RegisterResponderWithQuery("GET", apiURL, query, httpmock.NewStringResponder(500, ""))

		ti := urban.New(&http.Client{Transport: mt})

		_, err := ti.Define(ctx, phrase)
		assert.Error(t, err, "urban: ErrValidator: response error for https://api.urbandictionary.com/v0/define?term=something: unexpected status: 500")
	})

	t.Run("Unknown error", func(t *testing.T) {
		t.Parallel()
		mt := httpmockx.NewMockTransport(t)
		mt.RegisterResponderWithQuery("GET", apiURL, query, httpmock.NewStringResponder(418, ""))

		ti := urban.New(&http.Client{Transport: mt})

		_, err := ti.Define(ctx, phrase)
		assert.Error(t, err, "urban: ErrValidator: response error for https://api.urbandictionary.com/v0/define?term=something: unexpected status: 418")
	})

	t.Run("Bad JSON", func(t *testing.T) {
		t.Parallel()
		mt := httpmockx.NewMockTransport(t)
		mt.RegisterResponderWithQuery("GET", apiURL, query, httpmock.NewStringResponder(200, "}"))

		ti := urban.New(&http.Client{Transport: mt})

		_, err := ti.Define(ctx, phrase)
		assert.Error(t, err, "urban: ErrHandler: invalid character '}' looking for beginning of value")
	})
}
