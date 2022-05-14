package fakeirc

import (
	"context"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/hortbot/hortbot/internal/pkg/errgroupx"
	"github.com/jakebailey/irc"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"
)

const (
	defaultSleepDur = 50 * time.Millisecond
	sleepEnvVarName = "TEST_HELPER_SLEEP_DUR"
)

func getSleepDur() (time.Duration, error) {
	s, ok := os.LookupEnv(sleepEnvVarName)
	if ok {
		return time.ParseDuration(s)
	}

	return defaultSleepDur, nil
}

// Helper wraps the fake IRC server, providing convenience methods for managing messages.
type Helper struct {
	SleepDur time.Duration

	stopOnce sync.Once

	t *testing.T
	g *errgroupx.Group
	s *Server
}

// NewHelper creates a new Helper.
func NewHelper(ctx context.Context, t *testing.T, opts ...Option) *Helper { //nolint:thelper
	t.Helper()

	dur, err := getSleepDur()
	assert.NilError(t, err)

	server, err := Start(opts...)
	assert.NilError(t, err)
	assert.Assert(t, server != nil)

	return &Helper{
		SleepDur: dur,
		t:        t,
		g:        errgroupx.FromContext(ctx),
		s:        server,
	}
}

// CollectFromChannel collects all messages sent to the provided channel, asynchronously,
// returning a pointer to a slice which will contain the messages.
func (h *Helper) CollectFromChannel(ch <-chan *irc.Message) *[]*irc.Message {
	h.t.Helper()
	messages := []*irc.Message{}

	h.g.Go(func(ctx context.Context) error {
		for {
			select {
			case m, ok := <-ch:
				if !ok {
					return nil
				}

				m.Raw = ""
				messages = append(messages, m)

			case <-ctx.Done():
				return ctx.Err()
			}
		}
	})

	return &messages
}

// CollectSentToServer collects all messages sent to the server, asynchronously,
// returning a pointer to a slice which will contain the messages.
func (h *Helper) CollectSentToServer() *[]*irc.Message {
	h.t.Helper()
	return h.CollectFromChannel(h.ServerMessages())
}

// CollectFromConn collects all messages sent to a conn, asynchronously,
// returning a pointer to a slice which will contain the messages.
func (h *Helper) CollectFromConn(conn irc.Decoder) *[]*irc.Message {
	h.t.Helper()
	messages := []*irc.Message{}

	h.g.Go(func(_ context.Context) error {
		h.t.Helper()

		for {
			m := &irc.Message{}
			if err := conn.Decode(m); err != nil {
				err = ignoreClose(err)
				assert.NilError(h.t, err)
				return nil
			}

			m.Raw = ""
			messages = append(messages, m)
		}
	})

	return &messages
}

// ServerMessages returns the channel of messages sent to the server.
func (h *Helper) ServerMessages() <-chan *irc.Message {
	return h.s.Incoming()
}

// StopServer stops the fake IRC server.
func (h *Helper) StopServer() {
	h.t.Helper()
	_ = h.StopServerErr()
}

// StopServerErr stops the fake IRC server, returning the stop error.
func (h *Helper) StopServerErr() (err error) {
	h.t.Helper()

	h.stopOnce.Do(func() {
		h.t.Helper()
		err = h.s.Stop()
	})

	return err
}

// Wait waits for the server to stop.
func (h *Helper) Wait() {
	h.t.Helper()
	assert.NilError(h.t, h.g.Wait())
}

// Sleep sleeps for a small period of time.
func (h *Helper) Sleep() {
	h.t.Helper()
	time.Sleep(h.SleepDur)
}

// SendAsServer sends a message as the server, asserting that the send was successful.
func (h *Helper) SendAsServer(ctx context.Context, m *irc.Message) {
	h.t.Helper()
	assert.NilError(h.t, h.s.Send(ctx, m))
	h.Sleep()
}

// SendAsServerErr sends a message to the server, and returns the send error.
func (h *Helper) SendAsServerErr(ctx context.Context, m *irc.Message) error {
	h.t.Helper()
	defer h.Sleep()
	return h.s.Send(ctx, m)
}

// Addr returns the address of the server.
func (h *Helper) Addr() string {
	h.t.Helper()
	return h.s.Addr()
}

// Dial dials an IRC connection to the server.
func (h *Helper) Dial() irc.Conn {
	h.t.Helper()
	conn, err := h.s.Dial()
	assert.NilError(h.t, err)
	return conn
}

// CloseConn closes an IRC conn, asserting that it was successfully closed.
func (h *Helper) CloseConn(conn irc.Conn) {
	h.t.Helper()
	assert.NilError(h.t, ignoreClose(conn.Close()))
}

// SendWithConn sends a message via an IRC conn, asserts its success, and sleeps.
func (h *Helper) SendWithConn(conn irc.Encoder, m *irc.Message) {
	h.t.Helper()
	assert.NilError(h.t, conn.Encode(m))
	h.Sleep()
}

// AssertMessages asserts that the given messages match the expected messages.
func (h *Helper) AssertMessages(gotP *[]*irc.Message, want ...*irc.Message) {
	h.t.Helper()

	if gotP == nil {
		h.t.Fatal("nil gotP")
		panic("unreachable")
	}

	got := *gotP

	assert.Check(h.t, cmp.DeepEqual(want, got, cmpopts.EquateEmpty()))
}
