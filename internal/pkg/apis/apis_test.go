package apis_test

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/hortbot/hortbot/internal/pkg/apis"
	"gotest.tools/v3/assert"
)

func TestError(t *testing.T) {
	tests := []struct {
		err            *apis.Error
		m              string
		isNotFound     bool
		isServerError  bool
		isNotPermitted bool
	}{
		{
			err:        &apis.Error{StatusCode: 404},
			m:          "apis: status code 404",
			isNotFound: true,
		},
		{
			err:        &apis.Error{StatusCode: 404, Err: fmt.Errorf("we couldn't find your info")},
			m:          "apis: status code 404: we couldn't find your info",
			isNotFound: true,
		},
		{
			err:            &apis.Error{API: "service", StatusCode: 401},
			m:              "service: status code 401",
			isNotPermitted: true,
		},
		{
			err:            &apis.Error{API: "service", StatusCode: 403},
			m:              "service: status code 403",
			isNotPermitted: true,
		},
		{
			err: &apis.Error{API: "service", StatusCode: 451, Err: fmt.Errorf("censored")},
			m:   "service: status code 451: censored",
		},
		{
			err:           &apis.Error{API: "service", StatusCode: 500},
			m:             "service: status code 500",
			isServerError: true,
		},
		{
			err:           &apis.Error{API: "service", StatusCode: 501},
			m:             "service: status code 501",
			isServerError: true,
		},
	}

	for i, test := range tests {
		test := test
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			assert.Equal(t, test.err.Error(), test.m)
			assert.Equal(t, test.err.IsNotFound(), test.isNotFound)
			assert.Equal(t, test.err.IsServerError(), test.isServerError)
			assert.Equal(t, test.err.IsNotPermitted(), test.isNotPermitted)
		})
	}
}

func TestIsOK(t *testing.T) {
	assert.Assert(t, apis.IsOK(200))
	assert.Assert(t, apis.IsOK(204))
	assert.Assert(t, !apis.IsOK(100))
	assert.Assert(t, !apis.IsOK(404))
	assert.Assert(t, !apis.IsOK(451))
	assert.Assert(t, !apis.IsOK(500))
}
