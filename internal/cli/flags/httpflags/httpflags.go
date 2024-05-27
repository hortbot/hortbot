// Package httpflags provides HTTP client flags.
package httpflags

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/wader/filtertransport"
	"github.com/zikaeroh/ctxlog"
)

// HTTP contains HTTP client flags.
type HTTP struct {
	Timeout time.Duration `long:"http-timeout" env:"HB_HTTP_TIMEOUT" description:"HTTP client timeout"`

	UntrustedTimeout       time.Duration `long:"http-untrusted-timeout" env:"HB_HTTP_UNTRUSTED_TIMEOUT" description:"Untrusted HTTP client timeout"`
	UntrustedProxy         string        `long:"http-untrusted-proxy" env:"HB_HTTP_UNTRUSTED_PROXY" description:"Untrusted HTTP client proxy address"`
	UntrustedProxyUser     string        `long:"http-untrusted-proxy-user" env:"HB_HTTP_UNTRUSTED_PROXY_USER" description:"Untrusted HTTP client proxy user"`
	UntrustedProxyPassword string        `long:"http-untrusted-proxy-password" env:"HB_HTTP_UNTRUSTED_PROXY_PASSWORD" description:"Untrusted HTTP client proxy password"`
}

// Default contains the default flags. Make a copy of this, do not reuse.
var Default = HTTP{
	Timeout:          10 * time.Second,
	UntrustedTimeout: 2 * time.Second,
}

// Client returns a new HTTP client configured based on the flags.
func (h *HTTP) Client() *http.Client {
	return &http.Client{
		Timeout:   h.Timeout,
		Transport: filtertransport.DefaultTransport,
	}
}

// UntrustedClient returns a new HTTP client which can be used for untrusted HTTP requests.
func (h *HTTP) UntrustedClient(ctx context.Context) *http.Client {
	cli := &http.Client{
		Timeout: h.UntrustedTimeout,
	}

	if h.UntrustedProxy != "" {
		u := &url.URL{
			Scheme: "socks5",
			Host:   h.UntrustedProxy,
		}

		if h.UntrustedProxyUser != "" {
			if h.UntrustedProxyPassword == "" {
				u.User = url.User(h.UntrustedProxyUser)
			} else {
				u.User = url.UserPassword(h.UntrustedProxyUser, h.UntrustedProxyPassword)
			}
		}

		transport := (http.DefaultTransport).(*http.Transport).Clone()
		transport.Proxy = func(r *http.Request) (*url.URL, error) {
			// Hack to pre-filter the address before handing it to the proxy.

			// Similar to canonicalAddr in net/http.
			host := r.URL.Hostname()
			if v, err := idnaASCII(host); err == nil {
				host = v
			}

			port := r.URL.Port()
			if port == "" {
				switch r.URL.Scheme {
				case "http":
					port = "80"
				case "https":
					port = "443"
				default:
					return nil, fmt.Errorf("unknown scheme: %s", r.URL.Scheme)
				}
			}

			addr := net.JoinHostPort(host, port)

			tcpAddr, err := net.ResolveTCPAddr("tcp", addr)
			if err != nil {
				return nil, fmt.Errorf("resolving TCP address: %w", err)
			}

			if err := filtertransport.DefaultFilter(*tcpAddr); err != nil {
				return nil, fmt.Errorf("filtered address: %w", err)
			}

			return u, nil
		}

		cli.Transport = transport
	} else {
		ctxlog.Warn(ctx, "no proxy provided for untrusted HTTP client")
		cli.Transport = filtertransport.DefaultTransport
	}

	return cli
}
