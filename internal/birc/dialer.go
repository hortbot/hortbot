package birc

import (
	"context"
	"crypto/tls"
	"net"
	"strings"
)

// Twitch IRC addresses.
const (
	TwitchAddr    = "irc.chat.twitch.tv:6667"
	TwitchTLSAddr = "irc.chat.twitch.tv:6697"
)

// DefaultDialer is the default dialer used for conns, connecting to the Twitch
// IRC server securely.
var DefaultDialer = Dialer{
	Addr: TwitchTLSAddr,
}

// Dialer dials underlying TCP connections to IRC servers. The default value
// is valid for use.
type Dialer struct {
	// Addr is the IRC address to connect to, in hostname:port form.
	Addr string

	// Insecure will disable TLS if set to true.
	Insecure bool

	// TLSConfig is a TLS config to be used when connecting to the server.
	// If nil, the default will be used. If Insecure is true, this config
	// will not be used.
	TLSConfig *tls.Config

	// Dialer is the dialer used to connect to the IRC server. If unset, the
	// default will be used.
	Dialer *net.Dialer
}

// Dial dials a connection to a server.
func (d Dialer) Dial(ctx context.Context) (conn net.Conn, err error) {
	dialer := d.Dialer
	if dialer == nil {
		dialer = &net.Dialer{}
	}

	conn, err = dialer.DialContext(ctx, "tcp", d.Addr)
	if err != nil {
		return nil, err
	}

	if !d.Insecure {
		conn, err = upgradeToTLS(ctx, conn, d.Addr, d.TLSConfig)
		if err != nil {
			return nil, err
		}
	}

	return conn, nil
}

// See tls.DialWithDialer. Do this with a context by hand until the Go TLS
// library supports this: https://golang.org/issue/18482
//
// On an error, rawConn will be closed.
func upgradeToTLS(ctx context.Context, rawConn net.Conn, addr string, config *tls.Config) (conn *tls.Conn, err error) {
	defer func() {
		if err != nil {
			// The handshake error is the interesting one, so discard this.
			_ = rawConn.Close()
		}
	}()

	if config == nil || config.ServerName == "" {
		colonPos := strings.LastIndex(addr, ":")
		if colonPos == -1 {
			colonPos = len(addr)
		}
		hostname := addr[:colonPos]

		if config == nil {
			config = &tls.Config{}
		} else {
			config = config.Clone()
		}

		config.ServerName = hostname
	}

	conn = tls.Client(rawConn, config)

	// Buffer size of 1 to ensure the handshake goroutine exits and this
	// channel is garbage collected.
	errChan := make(chan error, 1)

	go func() {
		select {
		case errChan <- conn.Handshake():
		case <-ctx.Done():
		}
	}()

	select {
	case err = <-errChan:
		if err != nil {
			return nil, err
		}
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	return conn, nil
}
