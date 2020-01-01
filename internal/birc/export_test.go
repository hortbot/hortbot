package birc

import (
	"context"
	"net"

	"github.com/hortbot/hortbot/internal/birc/breq"
	"github.com/jakebailey/irc"
)

// Quit sends a QUIT to the IRC server, which may cause the client to
// disconnect.
//
// Exported for testing only.
func (c *Connection) Quit(ctx context.Context) error {
	return c.send(ctx, &irc.Message{Command: "QUIT"})
}

// Quit sends a QUIT message via one of the subconns. This will cause the IRC
// server to disconnect, so the subconn will exit.
//
// Exported for testing only.
func (p *Pool) Quit(ctx context.Context) error {
	return p.send(ctx, &irc.Message{Command: "QUIT"})
}

// ForceSubconn forces the creation of a subconn.
//
// Exported for testing only.
func (p *Pool) ForceSubconn(ctx context.Context) error {
	_, err := p.joinableConn(ctx, true)
	return err
}

// SendFrom sets a new sendFrom chan.
//
// Exported for testing only.
func (c *Connection) SendFrom(ch <-chan breq.Send) {
	c.sendFrom(ch)
}

// ConnDialer returns a dialer which always uses the given conn,
// rather than actually dialing an address.
func ConnDialer(conn net.Conn) *Dialer {
	return &Dialer{
		dial: func() (net.Conn, error) {
			return conn, nil
		},
	}
}
