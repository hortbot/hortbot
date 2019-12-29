package birc

import (
	"context"

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
