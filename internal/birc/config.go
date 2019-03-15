package birc

import (
	"fmt"
	"math/rand"
	"strings"
	"time"
)

// Config configures a Connection. Addr, Nick, and Pass must be specified.
type Config struct {
	// UserConfig configures the user that connects to the IRC server.
	UserConfig UserConfig

	// Dialer specifies the dialer to use to connect to the IRC server. If nil,
	// DefaultDialer will be used.
	Dialer *Dialer

	// InitialChannels specifies the list of channels the connection should
	// initially join once connected.
	//
	// These channels will all be joined via the same JOIN message, so if the
	// server has a JOIN rate limitation, it may be preferred to perform the
	// joins via the Join method manually.
	InitialChannels []string

	// Caps is a list of capabilities to declare to the server.
	Caps []string

	// RecvBuffer sets the buffer size for the received message buffer. A
	// buffer size of zero causes message receiving to be synchronous.
	//
	// This includes automatically handled PING/RECONNECT messages, so a slow
	// consumer may negatively impact the connection.
	RecvBuffer int
}

func (c *Config) Setup() {
	c.UserConfig.Setup()

	if c.Dialer == nil {
		c.Dialer = &DefaultDialer
	}

	if c.RecvBuffer < 0 {
		c.RecvBuffer = 0
	}
}

// UserConfig configures the user information for an IRC connection.
type UserConfig struct {
	// Nick is the nick to give when authenticating to the server.
	//
	// If Nick is empty, a random anonymous Twitch username will be used, and
	// the connection marked readonly.
	Nick string

	// Pass is the pass to give when authenticating to the server. If empty, it
	// will not be sent.
	Pass string

	// ReadOnly sets the connection to be read only. No messages may be sent
	// from this connection, other than control messages (like JOIN, PONG, etc).
	ReadOnly bool
}

func (u *UserConfig) Setup() {
	u.Nick = strings.ToLower(u.Nick)

	if u.Nick == "" {
		u.Nick = fmt.Sprintf("justinfan%d", rand.Int()) // #nosec G404
		u.Pass = ""
		u.ReadOnly = true
	}
}

type PoolConfig struct {
	// Config is the main IRC connection configuration.
	Config

	// MaxChannelsPerSubConn controls the maximum number of channels joined
	// per subconn.
	MaxChannelsPerSubConn int

	// JoinRate controls the number of channels the pool can join per second.
	// Set negative to disable.
	JoinRate float64

	// PruneInterval controls how often the pool prunes connections that are
	// not joined to any channels. Set negative to disable.
	PruneInterval time.Duration
}

func (p *PoolConfig) Setup() {
	p.Config.Setup()

	if p.MaxChannelsPerSubConn <= 0 {
		p.MaxChannelsPerSubConn = DefaultMaxChannelsPerSubConn
	}

	if p.JoinRate == 0 {
		p.JoinRate = DefaultJoinRate
	}

	if p.PruneInterval == 0 {
		p.PruneInterval = DefaultPruneInterval
	}
}
