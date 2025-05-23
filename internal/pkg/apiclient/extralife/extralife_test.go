package extralife_test

import (
	"errors"
	"net/http"
	"testing"

	"github.com/hortbot/hortbot/internal/pkg/apiclient"
	"github.com/hortbot/hortbot/internal/pkg/apiclient/extralife"
	"github.com/hortbot/hortbot/internal/pkg/httpmockx"
	"github.com/jarcoal/httpmock"
	"gotest.tools/v3/assert"
)

func TestGetDonationAmount(t *testing.T) {
	t.Parallel()
	mt := httpmockx.NewMockTransport(t)

	errTest := errors.New("test error")

	mt.RegisterResponder(
		"GET",
		"https://www.extra-life.org/api/participants/200",
		httpmock.NewStringResponder(200, `{"sumDonations": 123.45}`),
	)

	mt.RegisterResponder(
		"GET",
		"https://www.extra-life.org/api/participants/404",
		httpmock.NewStringResponder(404, `""`),
	)

	mt.RegisterResponder(
		"GET",
		"https://www.extra-life.org/api/participants/500",
		httpmock.NewStringResponder(500, `""`),
	)

	mt.RegisterResponder(
		"GET",
		"https://www.extra-life.org/api/participants/777",
		httpmock.NewStringResponder(200, `asdasd`),
	)

	mt.RegisterResponder(
		"GET",
		"https://www.extra-life.org/api/participants/999",
		httpmockx.ResponderFunc(func(_ *http.Request) (*http.Response, error) {
			return nil, errTest
		}),
	)

	t.Run("OK", func(t *testing.T) {
		t.Parallel()
		el := extralife.New(&http.Client{Transport: mt})

		amount, err := el.GetDonationAmount(t.Context(), 200)
		assert.NilError(t, err)
		assert.Equal(t, amount, float64(123.45))
	})

	t.Run("Not found", func(t *testing.T) {
		t.Parallel()
		el := extralife.New(&http.Client{Transport: mt})

		_, err := el.GetDonationAmount(t.Context(), 404)
		assert.Error(t, err, "extralife: ErrValidator: response error for https://www.extra-life.org/api/participants/404: unexpected status: 404")
	})

	t.Run("Server error", func(t *testing.T) {
		t.Parallel()
		el := extralife.New(&http.Client{Transport: mt})

		_, err := el.GetDonationAmount(t.Context(), 500)
		assert.Error(t, err, "extralife: ErrValidator: response error for https://www.extra-life.org/api/participants/500: unexpected status: 500")
	})

	t.Run("Decode error", func(t *testing.T) {
		t.Parallel()
		el := extralife.New(&http.Client{Transport: mt})

		_, err := el.GetDonationAmount(t.Context(), 777)
		e, ok := apiclient.AsError(err)
		if !ok {
			t.Fatalf("error has type %T", err)
			return
		}

		assert.Equal(t, e.API, "extralife")
		assert.ErrorContains(t, e.Err, "invalid character")
	})

	t.Run("Client error", func(t *testing.T) {
		t.Parallel()
		el := extralife.New(&http.Client{Transport: mt})

		_, err := el.GetDonationAmount(t.Context(), 999)
		assert.ErrorContains(t, err, errTest.Error())
	})
}
