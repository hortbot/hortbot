// Package httpmockx contains extensions to the httpmock package.
package httpmockx

import (
	"net/http"

	"github.com/hortbot/hortbot/internal/pkg/assertx"
	"github.com/jarcoal/httpmock"
)

// NewMockTransport creates a new MockTransport which will call t.Fatal on
// unmatched calls.
func NewMockTransport(t assertx.TestingT) *httpmock.MockTransport {
	t.Helper()

	fatal := func(args ...any) {
		t.Helper()
		t.Log(args...)
		t.FailNow()
	}

	mt := httpmock.NewMockTransport()
	mt.RegisterNoResponder(httpmock.NewNotFoundResponder(fatal))
	return mt
}

func ResponderFunc(f func(req *http.Request) (*http.Response, error)) httpmock.Responder {
	return func(r *http.Request) (*http.Response, error) {
		resp, err := f(r)
		if resp != nil {
			resp.Request = r
		}
		return resp, err
	}
}
