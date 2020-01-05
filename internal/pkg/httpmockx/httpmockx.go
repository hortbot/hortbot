package httpmockx

import (
	"testing"

	"github.com/jarcoal/httpmock"
)

// NewMockTransport creates a new MockTransport which will call t.Fatal on
// unmatched calls.
func NewMockTransport(t testing.TB) *httpmock.MockTransport {
	t.Helper()
	mt := httpmock.NewMockTransport()
	mt.RegisterNoResponder(httpmock.NewNotFoundResponder(t.Fatal))
	return mt
}
