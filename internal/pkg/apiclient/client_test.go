package apiclient_test

import (
	"errors"
	"strconv"
	"testing"

	"github.com/hortbot/hortbot/internal/pkg/apiclient"
	"gotest.tools/v3/assert"
)

func TestError(t *testing.T) {
	tests := []struct {
		err            *apiclient.Error
		m              string
		isNotFound     bool
		isServerError  bool
		isNotPermitted bool
	}{
		{
			err:        &apiclient.Error{StatusCode: 404},
			m:          "client: unexpected status: 404",
			isNotFound: true,
		},
		{
			err:        &apiclient.Error{StatusCode: 404, Err: errors.New("we couldn't find your info")},
			m:          "client: we couldn't find your info",
			isNotFound: true,
		},
		{
			err:            &apiclient.Error{API: "service", StatusCode: 401},
			m:              "service: unexpected status: 401",
			isNotPermitted: true,
		},
		{
			err:            &apiclient.Error{API: "service", StatusCode: 403},
			m:              "service: unexpected status: 403",
			isNotPermitted: true,
		},
		{
			err: &apiclient.Error{API: "service", StatusCode: 451, Err: errors.New("censored")},
			m:   "service: censored",
		},
		{
			err:           &apiclient.Error{API: "service", StatusCode: 500},
			m:             "service: unexpected status: 500",
			isServerError: true,
		},
		{
			err:           &apiclient.Error{API: "service", StatusCode: 501},
			m:             "service: unexpected status: 501",
			isServerError: true,
		},
	}

	for i, test := range tests {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			assert.Equal(t, test.err.Error(), test.m)
			assert.Equal(t, test.err.IsNotFound(), test.isNotFound)
			assert.Equal(t, test.err.IsServerError(), test.isServerError)
			assert.Equal(t, test.err.IsNotPermitted(), test.isNotPermitted)
		})
	}
}
