package fakeirc

import (
	"context"
	"crypto/tls"
	"errors"
	"io"
	"net"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/hortbot/hortbot/internal/pkg/errgroupx"
	"github.com/hortbot/hortbot/internal/pkg/ircx"
	"github.com/jakebailey/irc"
)

// ChannelTag is a tag key which can be used to direct messages at specific
// channels.
const ChannelTag = "test-channel"

// ErrStopped is returned by Server.Send if the server has been stopped.
var ErrStopped = errors.New("fakeirc: server is stopped")

// Server is a fake IRC server for use in testing.
type Server struct {
	stopChan chan struct{}
	stopped  uint32

	tlsConfig *tls.Config

	listener net.Listener

	g      *errgroupx.Group
	cancel func()

	recvChan chan *irc.Message
	sendChan chan *irc.Message

	conns   map[irc.Conn]map[string]bool
	connsMu sync.RWMutex

	pong        bool
	recordPings bool
}

// Start starts a new fake IRC server.
func Start(opts ...Option) (*Server, error) {
	s := &Server{
		stopChan: make(chan struct{}),
		recvChan: make(chan *irc.Message),
		sendChan: make(chan *irc.Message),
		conns:    make(map[irc.Conn]map[string]bool),
		pong:     true,
	}

	for _, opt := range opts {
		opt(s)
	}

	var err error

	if s.tlsConfig == nil {
		s.listener, err = net.Listen("tcp", "localhost:0")
	} else {
		s.listener, err = tls.Listen("tcp", "localhost:0", s.tlsConfig)
	}

	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(context.Background())
	s.g = errgroupx.FromContext(ctx)
	s.cancel = cancel

	s.g.Go(s.accepter)
	s.g.Go(s.sender)

	return s, nil
}

// Stop stops the server and waits for all worker goroutines to exit.
func (s *Server) Stop() error {
	if atomic.SwapUint32(&s.stopped, 1) == 0 {
		defer close(s.recvChan)

		// No sync.Once; This package is for testing so a panic is more useful.
		close(s.stopChan)
		s.listener.Close()

		// TODO: Fragile, replace
		time.Sleep(10 * time.Millisecond)
		s.g.Stop()
	}

	return s.g.WaitIgnoreStop()
}

// Incoming returns the channel of messages being sent to the server from clients.
func (s *Server) Incoming() <-chan *irc.Message {
	return s.recvChan
}

// Send sends a message as the server to all relevent clients.
func (s *Server) Send(ctx context.Context, m *irc.Message) error {
	select {
	case s.sendChan <- m:
		return nil
	case <-s.stopChan:
		return ErrStopped
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Addr gets the address the server is listening on.
func (s *Server) Addr() string {
	return s.listener.Addr().String()
}

// Dial dials a new IRC connnection to the server, handling TLS if needed.
func (s *Server) Dial() (irc.Conn, error) {
	if s.tlsConfig == nil {
		return irc.BaseDial(s.Addr())
	}

	conn, err := tls.Dial("tcp", s.Addr(), s.tlsConfig)
	return irc.NewBaseConn(conn), err
}

func (s *Server) accepter(ctx context.Context) error {
	defer s.listener.Close()

	for {
		rawConn, err := s.listener.Accept()
		if err != nil {
			return ignoreClose(err)
		}

		conn := irc.NewBaseConn(rawConn)

		s.connsMu.Lock()
		s.conns[conn] = make(map[string]bool)
		s.connsMu.Unlock()

		s.g.Go(func(ctx context.Context) error {
			return s.handle(ctx, conn)
		})
	}
}

func (s *Server) sender(ctx context.Context) error {
	for {
		select {
		case m := <-s.sendChan:
			if err := s.send(m, nil); err != nil {
				return err
			}

		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func (s *Server) send(m *irc.Message, origin irc.Conn) error {
	var channel string
	var filter bool

	if m.Tags != nil {
		channel, filter = m.Tags[ChannelTag]
	} else if m.Command == "PRIVMSG" && len(m.Params) > 0 {
		channel = m.Params[0]
		filter = true
	}

	s.connsMu.RLock()
	defer s.connsMu.RUnlock()

	for conn, channels := range s.conns {
		if conn == origin {
			continue
		}

		if filter && !channels[channel] {
			continue
		}

		if err := ignoreClose(conn.Encode(m)); err != nil {
			return err
		}
	}

	return nil
}

func (s *Server) handle(ctx context.Context, conn irc.Conn) error {
	stop := make(chan struct{})
	defer close(stop)

	defer func() {
		s.connsMu.Lock()
		delete(s.conns, conn)
		defer s.connsMu.Unlock()
	}()

	defer conn.Close()

	s.g.Go(func(ctx context.Context) error {
		select {
		case <-stop:
		case <-ctx.Done():
		}
		return ignoreClose(conn.Close())
	})

	for {
		m := &irc.Message{}
		if err := conn.Decode(m); err != nil {
			return ignoreClose(err)
		}

		switch m.Command {
		case "JOIN", "PART":
			s.joinPart(conn, m)
		case "PRIVMSG":
			if err := s.send(m, conn); err != nil {
				return err
			}
		case "PING":
			if s.pong {
				pong := *m
				pong.Command = "PONG"
				if err := conn.Encode(&pong); err != nil {
					return err
				}
			}

			if !s.recordPings {
				continue
			}
		}

		select {
		case s.recvChan <- m:
			// Do nothing.
		case <-ctx.Done():
			return ctx.Err()
		}

		if m.Command == "QUIT" {
			return nil
		}
	}
}

func (s *Server) joinPart(conn irc.Conn, m *irc.Message) {
	s.connsMu.Lock()
	defer s.connsMu.Unlock()

	for _, c := range m.Params {
		if m.Command == "JOIN" {
			s.conns[conn][c] = true
		} else {
			delete(s.conns[conn], c)
		}
	}
}

// TagChannel tags a message with a channel name. It modifies the message
// in-place, but returns the message for convenience.
func TagChannel(m *irc.Message, channel string) *irc.Message {
	channel = ircx.NormalizeChannel(channel)

	if m.Tags == nil {
		m.Tags = map[string]string{ChannelTag: channel}
	} else {
		m.Tags[ChannelTag] = channel
	}

	return m
}

// Option configures the server.
type Option func(s *Server)

// TLS enables TLS for the server given a TLS config.
func TLS(config *tls.Config) Option {
	return func(s *Server) {
		s.tlsConfig = config
	}
}

// Pong controls whether or not the server responds to PINGS with PONGS.
func Pong(enable bool) Option {
	return func(s *Server) {
		s.pong = enable
	}
}

// RecordPings enables recording of incoming pings the incoming channel.
func RecordPings(enable bool) Option {
	return func(s *Server) {
		s.recordPings = enable
	}
}

func ignoreClose(err error) error {
	switch {
	case err == nil:
	case err == io.EOF:
	case strings.Contains(err.Error(), "use of closed"):
	default:
		return err
	}

	return nil
}
