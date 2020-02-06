// Package httpflags provides HTTP client flags.
package httpflags

import (
	"net/http"
	"time"
)

// HTTP contains HTTP client flags.
type HTTP struct {
	Timeout time.Duration `long:"http-timeoout" env:"HB_HTTP_TIMEOUT" description:"HTTP client timeout"`
}

// Default contains the default flags. Make a copy of this, do not reuse.
var Default = HTTP{
	Timeout: 10 * time.Second,
}

// Client returns a new HTTP client configured based on the flags.
func (h *HTTP) Client() *http.Client {
	return &http.Client{
		Timeout: h.Timeout,
	}
}
