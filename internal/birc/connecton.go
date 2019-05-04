package birc

import (
	"context"
	"net"
	"sort"
	"sync"

	"github.com/hortbot/hortbot/internal/birc/breq"
	"github.com/hortbot/hortbot/internal/ctxlog"
	"github.com/hortbot/hortbot/internal/x/errgroupx"
	"github.com/hortbot/hortbot/internal/x/ircx"
	"github.com/jakebailey/irc"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

var (
	// ErrConnectionClosed is returned when a Connection is closed, so a message
	// cannot be sent.
	ErrConnectionClosed = errors.New("birc: connection closed")

	// ErrReconnect is returned when the connection closes as the server has
	// requested a reconnect.
	ErrReconnect = errors.New("birc: server asked for reconnect")

	// ErrReadOnly is returned when a read only connection is used to send a message.
	ErrReadOnly = errors.New("birc: connection is marked read only")
)

// Connection manages a single connection to an IRC server.
type Connection struct {
	config *Config

	conn irc.Conn

	recvChan     chan *irc.Message
	sendChan     chan breq.Send
	sendFromChan chan (<-chan breq.Send)

	closeOnce sync.Once
	closed    chan struct{}
	closeErr  error

	joinedMu sync.RWMutex
	joined   map[string]bool

	ready chan struct{}
}

// NewConnection creates a new Connection.
func NewConnection(config Config) *Connection {
	config.Setup()
	return newConnection(&config)
}

func newConnection(config *Config) *Connection {
	return &Connection{
		config:       config,
		recvChan:     make(chan *irc.Message, config.RecvBuffer),
		sendChan:     make(chan breq.Send),
		sendFromChan: make(chan (<-chan breq.Send), 1),
		closed:       make(chan struct{}),
		joined:       make(map[string]bool),
		ready:        make(chan struct{}),
	}
}

// Run starts the connection and returns when the connection is closed.
//
// Once this function has returned, the connection cannot be reused.
func (c *Connection) Run(ctx context.Context) (err error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	var nconn net.Conn
	nconn, err = c.config.Dialer.Dial(ctx)
	if err != nil {
		return errors.Wrap(err, "dialing connection")
	}

	g := errgroupx.FromContext(ctx)

	c.conn = irc.NewBaseConn(nconn)
	defer func() {
		cerr := c.Close()
		if err == nil && cerr != nil {
			err = cerr
		}
	}()

	g.Go(func(ctx context.Context) error {
		select {
		case <-ctx.Done():
		case <-c.closed:
		}
		return c.Close()
	})

	if c.config.UserConfig.Pass != "" {
		if err := c.conn.Encode(ircx.Pass(c.config.UserConfig.Pass)); err != nil {
			return errors.Wrap(err, "sending pass")
		}
	}

	if err := c.conn.Encode(ircx.Nick(c.config.UserConfig.Nick)); err != nil {
		return errors.Wrap(err, "sending nick")
	}

	if len(c.config.Caps) > 0 {
		if err := c.conn.Encode(ircx.CapReq(c.config.Caps...)); err != nil {
			return errors.Wrap(err, "sending capabilities")
		}
	}

	if len(c.config.InitialChannels) > 0 {
		initChannels := make([]string, 0, len(c.config.InitialChannels))

		for _, ch := range c.config.InitialChannels {
			ch = ircx.NormalizeChannel(ch)
			if ch == "" {
				continue
			}

			initChannels = append(initChannels, ch)
			c.joined[ch] = true
		}

		if err := c.conn.Encode(ircx.Join(initChannels...)); err != nil {
			return errors.Wrap(err, "joining initial channels")
		}
	}

	g.Go(c.reciever)
	g.Go(c.sender)

	close(c.ready)

	return g.Wait() // TODO: Convert io.EOF to something else?
}

// Close closes the IRC connection. This function is safe to call more than
// once and safe for concurrent use. All calls following the first will return
// the same error.
func (c *Connection) Close() error {
	c.closeOnce.Do(func() {
		if c.conn != nil {
			c.closeErr = c.conn.Close()
		}
		close(c.closed)
	})
	return c.closeErr
}

func (c *Connection) WaitUntilReady(ctx context.Context) error {
	select {
	case <-c.ready:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// SendMessage sends a PRIVMSG to the specified target.
func (c *Connection) SendMessage(ctx context.Context, target, message string) error {
	if c.config.UserConfig.ReadOnly {
		return ErrReadOnly
	}
	return c.send(ctx, ircx.PrivMsg(target, message))
}

// Incoming returns a channel which is sent incoming messages. When the
// connection is closed this channel will also be closed. Note that the
// returned channel is shared among all callers; only one receiver will
// be able to receive any given message.
func (c *Connection) Incoming() <-chan *irc.Message {
	return c.recvChan
}

func (c *Connection) reciever(ctx context.Context) error {
	defer close(c.recvChan)

	logger := ctxlog.FromContext(ctx)

	for {
		m := &irc.Message{}
		if err := c.conn.Decode(m); err != nil {
			if pe, ok := err.(*irc.ParseError); ok {
				logger.Warn("recieved bad message from IRC server, ignoring", zap.Error(pe))
				continue
			}
			return err
		}

		if m.Command == "PING" {
			pong := *m
			pong.Command = "PONG"

			go func() {
				if err := c.send(ctx, &pong); err != nil {
					logger.Error("error sending pong", zap.Error(err))
				}
			}()
		}

		select {
		case c.recvChan <- m:
			// Do nothing.
		case <-ctx.Done():
			return ctx.Err()
		}

		// The connection should be closed by the server after this message,
		// but we can return early here to attempt to prevent any future uses
		// of this connection that would just fail.
		if m.Command == "RECONNECT" {
			return ErrReconnect
		}
	}
}

func (c *Connection) sender(ctx context.Context) error {
	logger := ctxlog.FromContext(ctx)

	var sendFrom <-chan breq.Send

	for {
		var req breq.Send

		select {
		case req = <-c.sendChan:
		case sfReq, ok := <-sendFrom:
			if !ok {
				sendFrom = nil
				continue
			}
			req = sfReq

		case sendFrom = <-c.sendFromChan:
			continue

		case <-ctx.Done():
			return ctx.Err()
		}

		err := c.conn.Encode(req.M)
		req.Finish(err)

		if err != nil {
			return errors.Wrap(err, "sending to conn")
		}

		logger.Debug("sent", zap.Any("m", req.M))
	}
}

func (c *Connection) send(ctx context.Context, m *irc.Message) error {
	return breq.NewSend(m).Do(ctx, c.sendChan, c.closed, ErrConnectionClosed)
}

// sendFrom sets an extra channel the sender will send requests from. This
// is safe to call before Run.
func (c *Connection) sendFrom(ch <-chan breq.Send) {
	c.sendFromChan <- ch
}

// Join instructs the connection to join the specified channels.
//
// Note that even if an error occurs, the connection's state will be updated
// such that it appears that the channels were parted.
func (c *Connection) Join(ctx context.Context, channels ...string) error {
	return c.doJoinPart(ctx, true, channels...)
}

// Joined returns a list of the joined channels. It is safe for concurrent use,
// and is available even after the connection has closed. This list may not be
//
func (c *Connection) Joined() []string {
	c.joinedMu.RLock()
	defer c.joinedMu.RUnlock()

	joined := make([]string, 0, len(c.joined))

	for ch := range c.joined {
		joined = append(joined, ch)
	}

	sort.Strings(joined)

	return joined
}

// IsJoined returns true if the specified channel has been joined.
func (c *Connection) IsJoined(channel string) bool {
	channel = ircx.NormalizeChannel(channel)

	if channel == "" {
		return false
	}

	c.joinedMu.RLock()
	defer c.joinedMu.RUnlock()
	return c.joined[channel]
}

// NumJoined returns the number of joined channels.
func (c *Connection) NumJoined() int {
	c.joinedMu.RLock()
	defer c.joinedMu.RUnlock()
	return len(c.joined)
}

// Part instructs the connection to part with the specified channels.
//
// Note that even if an error occurs, the connection's state will be updated
// such that it appears that the channels were parted.
func (c *Connection) Part(ctx context.Context, channels ...string) error {
	return c.doJoinPart(ctx, false, channels...)
}

func (c *Connection) doJoinPart(ctx context.Context, join bool, channels ...string) error {
	if len(channels) == 0 {
		return nil
	}

	c.joinedMu.Lock()
	defer c.joinedMu.Unlock()

	changes := make([]string, 0, len(channels))

	for _, ch := range channels {
		ch = ircx.NormalizeChannel(ch)
		if ch == "" {
			continue
		}

		if c.joined[ch] == join {
			continue
		}

		if join {
			c.joined[ch] = true
		} else {
			delete(c.joined, ch)
		}

		changes = append(changes, ch)
	}

	if len(changes) == 0 {
		return nil
	}

	var m *irc.Message

	if join {
		m = ircx.Join(changes...)
	} else {
		m = ircx.Part(changes...)
	}

	return c.send(ctx, m)
}

// Ping sends a PING message to the target. If any response is received, it
// will be delivered like any other message.
func (c *Connection) Ping(ctx context.Context, target string) error {
	return c.send(ctx, &irc.Message{
		Command:  "PING",
		Trailing: target,
	})
}

// Quit sends a QUIT to the IRC server, which may cause the client to
// disconnect.
func (c *Connection) Quit(ctx context.Context) error {
	return c.send(ctx, &irc.Message{Command: "QUIT"})
}
