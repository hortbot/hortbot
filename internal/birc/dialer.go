package birc

import (
	"context"
	"crypto/tls"
	"net"
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

	conn net.Conn
}

// Dial dials a connection to a server.
func (d Dialer) Dial(ctx context.Context) (conn net.Conn, err error) {
	if d.conn != nil {
		return d.conn, nil
	}

	dialer := d.Dialer
	if dialer == nil {
		dialer = &net.Dialer{}
	}

	if !d.Insecure {
		// TODO: Use DialContext once it exists. https://golang.org/issue/18482
		return tls.DialWithDialer(dialer, "tcp", d.Addr, d.TLSConfig)
	}

	return dialer.DialContext(ctx, "tcp", d.Addr)
}
