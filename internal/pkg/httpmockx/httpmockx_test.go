package httpmockx_test

import (
	"net/http"
	"testing"

	"github.com/hortbot/hortbot/internal/pkg/assertx/assertxmocks"
	"github.com/hortbot/hortbot/internal/pkg/httpmockx"
	"gotest.tools/v3/assert"
)

func TestNewMockTransport(t *testing.T) {
	fake := &assertxmocks.TestingTMock{
		HelperFunc:  func() {},
		LogFunc:     func(args ...interface{}) {},
		FailNowFunc: func() {},
	}

	mt := httpmockx.NewMockTransport(fake)
	assert.Assert(t, mt != nil)
	assert.Equal(t, len(fake.HelperCalls()), 1)

	client := &http.Client{
		Transport: mt,
	}

	_, err := client.Get("http://example.org") //nolint
	assert.ErrorContains(t, err, "not found")

	assert.Assert(t, len(fake.HelperCalls()) > 1)
	assert.Assert(t, len(fake.LogCalls()) > 0)
	assert.Assert(t, len(fake.FailNowCalls()) > 0)
}
