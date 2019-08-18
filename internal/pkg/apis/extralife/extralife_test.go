package extralife_test

import (
	"errors"
	"net/http"
	"testing"

	"github.com/hortbot/hortbot/internal/pkg/apis/extralife"
	"github.com/jarcoal/httpmock"
	"gotest.tools/v3/assert"
)

func TestGetDonationAmount(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	errTest := errors.New("test error")

	httpmock.RegisterResponder(
		"GET",
		"https://www.extra-life.org/api/participants/200",
		httpmock.NewStringResponder(200, `{"sumDonations": 123.45}`),
	)

	httpmock.RegisterResponder(
		"GET",
		"https://www.extra-life.org/api/participants/404",
		httpmock.NewStringResponder(404, `""`),
	)

	httpmock.RegisterResponder(
		"GET",
		"https://www.extra-life.org/api/participants/500",
		httpmock.NewStringResponder(500, `""`),
	)

	httpmock.RegisterResponder(
		"GET",
		"https://www.extra-life.org/api/participants/777",
		httpmock.NewStringResponder(200, `asdasd`),
	)

	httpmock.RegisterResponder(
		"GET",
		"https://www.extra-life.org/api/participants/999",
		func(_ *http.Request) (*http.Response, error) {
			return nil, errTest
		},
	)

	t.Run("OK", func(t *testing.T) {
		el := extralife.New()

		amount, err := el.GetDonationAmount(200)
		assert.NilError(t, err)
		assert.Equal(t, amount, float64(123.45))
	})

	t.Run("Not found", func(t *testing.T) {
		el := extralife.New()

		_, err := el.GetDonationAmount(404)
		assert.Equal(t, err, extralife.ErrNotFound)
	})

	t.Run("Server error", func(t *testing.T) {
		el := extralife.New()

		_, err := el.GetDonationAmount(500)
		assert.Equal(t, err, extralife.ErrServerError)
	})

	t.Run("Decode error", func(t *testing.T) {
		el := extralife.New()

		_, err := el.GetDonationAmount(777)
		assert.Equal(t, err, extralife.ErrServerError)
	})

	t.Run("Client error", func(t *testing.T) {
		el := extralife.New()

		_, err := el.GetDonationAmount(999)
		assert.ErrorContains(t, err, errTest.Error())
	})
}

func TestGetDonationAmountWithClient(t *testing.T) {
	mt := httpmock.NewMockTransport()

	cli := &http.Client{
		Transport: mt,
	}

	mt.RegisterResponder(
		"GET",
		"https://www.extra-life.org/api/participants/200",
		httpmock.NewStringResponder(200, `{"sumDonations": 123.45}`),
	)

	el := extralife.New(extralife.HTTPClient(cli))

	amount, err := el.GetDonationAmount(200)
	assert.NilError(t, err)
	assert.Equal(t, amount, float64(123.45))
}
