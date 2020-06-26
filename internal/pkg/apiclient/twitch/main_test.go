package twitch

import (
	"errors"
	"net/http"
	"os"
	"testing"

	"github.com/jarcoal/httpmock"
)

func TestMain(m *testing.M) {
	httpmock.Activate()
	httpmock.RegisterNoResponder(func(*http.Request) (*http.Response, error) {
		return nil, errors.New("illegal attempt to use default HTTP transport")
	})
	code := m.Run()
	httpmock.DeactivateAndReset()

	os.Exit(code)
}
