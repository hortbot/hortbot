// Package httpflags provides HTTP client flags.
package httpflags

import (
	"net/http"
	"time"
)

type HTTP struct {
	Timeout time.Duration `long:"http-timeoout" env:"HB_HTTP_TIMEOUT" description:"HTTP client timeout"`
}

var DefaultHTTP = HTTP{
	Timeout: 10 * time.Second,
}

func (h *HTTP) Client() *http.Client {
	return &http.Client{
		Timeout: h.Timeout,
	}
}
