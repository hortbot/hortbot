// Package httpflags provides HTTP client flags.
package httpflags

import (
	"context"
	"net"
	"net/http"
	"time"

	"github.com/hortbot/hortbot/internal/pkg/ctxlog"
	"github.com/wader/filtertransport"
	"go.uber.org/zap"
	"golang.org/x/net/proxy"
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
		var auth *proxy.Auth

		if h.UntrustedProxyUser != "" {
			auth = &proxy.Auth{User: h.UntrustedProxyUser, Password: h.UntrustedProxyPassword}
		}

		// TODO: safeDialer performs name resolution locally rather than the proxy. Is this safe?
		dialer, err := proxy.SOCKS5("tcp", h.UntrustedProxy, auth, safeDialer)
		if err != nil {
			ctxlog.Fatal(ctx, "error creating SOCKS5 proxy dialer", zap.Error(err))
		}

		transport := filtertransport.DefaultTransport.Clone()

		//lint:ignore SA1019 As a backup in case DialContext is nil.
		transport.Dial = dialer.Dial //nolint

		if ctxDialer, ok := dialer.(proxy.ContextDialer); ok {
			transport.DialContext = ctxDialer.DialContext
		} else {
			transport.DialContext = nil
		}

		cli.Transport = transport
	} else {
		ctxlog.Warn(ctx, "no proxy provided for untrusted HTTP client")
		cli.Transport = filtertransport.DefaultTransport
	}

	return cli
}

type filteredDialer struct{}

var (
	netDialer  net.Dialer
	safeDialer *filteredDialer
	_          proxy.Dialer        = safeDialer
	_          proxy.ContextDialer = safeDialer
)

func (*filteredDialer) Dial(network, addr string) (c net.Conn, err error) {
	return filtertransport.FilterDial(context.Background(), network, addr, filtertransport.DefaultFilter, netDialer.DialContext)
}

func (*filteredDialer) DialContext(ctx context.Context, network, addr string) (net.Conn, error) {
	return filtertransport.FilterDial(ctx, network, addr, filtertransport.DefaultFilter, netDialer.DialContext)
}
