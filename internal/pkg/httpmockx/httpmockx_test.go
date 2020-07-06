package httpmockx_test

import (
	"net/http"
	"testing"

	"github.com/hortbot/hortbot/internal/pkg/assertx/assertxfakes"
	"github.com/hortbot/hortbot/internal/pkg/httpmockx"
	"gotest.tools/v3/assert"
)

func TestNewMockTransport(t *testing.T) {
	fake := &assertxfakes.FakeTestingT{}

	mt := httpmockx.NewMockTransport(fake)
	assert.Assert(t, mt != nil)
	assert.Equal(t, fake.HelperCallCount(), 1)

	client := &http.Client{
		Transport: mt,
	}

	_, err := client.Get("http://example.org") //nolint
	assert.ErrorContains(t, err, "not found")

	assert.Assert(t, fake.HelperCallCount() > 1)
	assert.Assert(t, fake.LogCallCount() > 0)
	assert.Assert(t, fake.FailNowCallCount() > 0)
}
