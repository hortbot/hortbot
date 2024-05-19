package extralife_test

import (
	"context"
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
		func(_ *http.Request) (*http.Response, error) {
			return nil, errTest
		},
	)

	t.Run("OK", func(t *testing.T) {
		el := extralife.New(&http.Client{Transport: mt})

		amount, err := el.GetDonationAmount(context.Background(), 200)
		assert.NilError(t, err)
		assert.Equal(t, amount, float64(123.45))
	})

	t.Run("Not found", func(t *testing.T) {
		el := extralife.New(&http.Client{Transport: mt})

		_, err := el.GetDonationAmount(context.Background(), 404)
		assert.DeepEqual(t, err, apiclient.NewStatusError("extralife", 404))
	})

	t.Run("Server error", func(t *testing.T) {
		el := extralife.New(&http.Client{Transport: mt})

		_, err := el.GetDonationAmount(context.Background(), 500)
		assert.DeepEqual(t, err, apiclient.NewStatusError("extralife", 500))
	})

	t.Run("Decode error", func(t *testing.T) {
		el := extralife.New(&http.Client{Transport: mt})

		_, err := el.GetDonationAmount(context.Background(), 777)
		e, ok := apiclient.AsError(err)
		if !ok {
			t.Fatalf("error has type %T", err)
			return
		}

		assert.Equal(t, e.API, "extralife")
		assert.ErrorContains(t, e.Err, "invalid character")
	})

	t.Run("Client error", func(t *testing.T) {
		el := extralife.New(&http.Client{Transport: mt})

		_, err := el.GetDonationAmount(context.Background(), 999)
		assert.ErrorContains(t, err, errTest.Error())
	})
}
