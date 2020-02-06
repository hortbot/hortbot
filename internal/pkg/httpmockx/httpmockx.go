// Package httpmockx contains extensions to the httpmock package.
package httpmockx

import (
	"github.com/hortbot/hortbot/internal/pkg/assertx"
	"github.com/jarcoal/httpmock"
)

// NewMockTransport creates a new MockTransport which will call t.Fatal on
// unmatched calls.
func NewMockTransport(t assertx.TestingT) *httpmock.MockTransport {
	t.Helper()

	fatal := func(args ...interface{}) {
		t.Helper()
		t.Log(args...)
		t.FailNow()
	}

	mt := httpmock.NewMockTransport()
	mt.RegisterNoResponder(httpmock.NewNotFoundResponder(fatal))
	return mt
}
