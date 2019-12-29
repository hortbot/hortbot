package birc_test

import (
	"context"
	"fmt"
	"io"
	"testing"
	"time"

	"github.com/hortbot/hortbot/internal/birc"
	"github.com/hortbot/hortbot/internal/birc/fakeirc"
	"github.com/hortbot/hortbot/internal/pkg/ircx"
	"github.com/jakebailey/irc"
	"gotest.tools/v3/assert"
)

func TestConnectionUnused(t *testing.T) {
	c := birc.NewConnection(birc.Config{})
	assert.Assert(t, c != nil)
	assert.NilError(t, c.Close())
}

func TestConnectionDialError(t *testing.T) {
	ctx, cancel := testContext()
	defer cancel()

	c := birc.NewConnection(birc.Config{
		Dialer: &birc.Dialer{
			Addr: "localhost:0",
		},
	})

	assert.ErrorContains(t, c.Run(ctx), "connection refused")
}

func TestConnectionBasic(t *testing.T) {
	doTestSecureInsecure(t, func(ctx context.Context, t *testing.T, h *fakeirc.Helper, d birc.Dialer, sm <-chan *irc.Message) {
		serverMessages := h.CollectSentToServer()

		c := birc.Config{
			UserConfig: birc.UserConfig{
				Nick: "nick",
				Pass: "pass",
			},
			Dialer: &d,
		}

		connErr := make(chan error, 1)
		conn := birc.NewConnection(c)

		go func() {
			connErr <- conn.Run(ctx)
		}()

		clientMessages := h.CollectFromChannel(conn.Incoming())

		assert.NilError(t, conn.WaitUntilReady(ctx))
		h.Sleep()

		assert.NilError(t, conn.SendMessage(ctx, "#foobar", "test"))
		h.SendAsServer(ctx, &irc.Message{Command: "PING"})

		h.Sleep()
		h.Sleep()
		h.Sleep()

		quitErr := conn.Quit(ctx)
		if quitErr != birc.ErrConnectionClosed {
			assert.NilError(t, quitErr)
		}

		assert.Equal(t, io.EOF, errFromErrChan(ctx, connErr))

		h.StopServer()
		h.Wait()

		h.AssertMessages(serverMessages,
			ircx.Pass("pass"),
			ircx.Nick("nick"),
			ircx.PrivMsg("#foobar", "test"),
			&irc.Message{Command: "PONG"},
			ircx.Quit(),
		)

		h.AssertMessages(clientMessages,
			&irc.Message{Command: "PING"},
		)

		assert.Assert(t, conn.NumJoined() == 0)
		assert.Assert(t, conn.IsJoined("#foobar") == false)
	})
}

func TestConnectionJoinQuit(t *testing.T) {
	doTest(t, func(ctx context.Context, t *testing.T, h *fakeirc.Helper, d birc.Dialer, sm <-chan *irc.Message) {
		serverMessages := h.CollectSentToServer()

		c := birc.Config{
			UserConfig: birc.UserConfig{
				Nick: "nick",
				Pass: "pass",
			},
			Dialer: &d,
		}

		connErr := make(chan error, 1)
		conn := birc.NewConnection(c)

		go func() {
			connErr <- conn.Run(ctx)
		}()

		clientMessages := h.CollectFromChannel(conn.Incoming())

		assert.NilError(t, conn.WaitUntilReady(ctx))
		h.Sleep()

		assert.Assert(t, conn.NumJoined() == 0)
		assert.Assert(t, !conn.IsJoined("#foobar"))
		assert.DeepEqual(t, []string{}, conn.Joined())

		assert.NilError(t, conn.Join(ctx, "#foobar"))
		assert.Assert(t, conn.NumJoined() == 1)
		assert.Assert(t, conn.IsJoined("#foobar"))
		assert.DeepEqual(t, []string{"#foobar"}, conn.Joined())

		quitErr := conn.Quit(ctx)
		if quitErr != birc.ErrConnectionClosed {
			assert.NilError(t, quitErr)
		}

		assert.Equal(t, io.EOF, errFromErrChan(ctx, connErr))

		h.StopServer()
		h.Wait()

		h.AssertMessages(serverMessages,
			ircx.Pass("pass"),
			ircx.Nick("nick"),
			ircx.Join("#foobar"),
			ircx.Quit(),
		)

		h.AssertMessages(clientMessages)

		assert.Assert(t, conn.NumJoined() == 1)
		assert.Assert(t, conn.IsJoined("#foobar") == true)
	})
}

func TestConnectionJoinPart(t *testing.T) {
	doTest(t, func(ctx context.Context, t *testing.T, h *fakeirc.Helper, d birc.Dialer, sm <-chan *irc.Message) {
		serverMessages := h.CollectSentToServer()

		c := birc.Config{
			UserConfig: birc.UserConfig{
				Nick: "nick",
				Pass: "pass",
			},
			Dialer: &d,
		}

		connErr := make(chan error, 1)
		conn := birc.NewConnection(c)

		go func() {
			connErr <- conn.Run(ctx)
		}()

		clientMessages := h.CollectFromChannel(conn.Incoming())

		assert.NilError(t, conn.WaitUntilReady(ctx))
		h.Sleep()

		assert.Assert(t, conn.NumJoined() == 0)
		assert.Assert(t, !conn.IsJoined("#foobar"))
		assert.DeepEqual(t, []string{}, conn.Joined())

		assert.NilError(t, conn.Join(ctx, "#foobar"))
		assert.Assert(t, conn.NumJoined() == 1)
		assert.Assert(t, conn.IsJoined("#foobar"))
		assert.DeepEqual(t, []string{"#foobar"}, conn.Joined())

		assert.NilError(t, conn.Part(ctx))
		assert.Assert(t, conn.NumJoined() == 1)
		assert.Assert(t, conn.IsJoined("#foobar"))
		assert.DeepEqual(t, []string{"#foobar"}, conn.Joined())

		assert.NilError(t, conn.Part(ctx, "#foobar"))
		assert.Assert(t, conn.NumJoined() == 0)
		assert.Assert(t, !conn.IsJoined("#foobar"))
		assert.DeepEqual(t, []string{}, conn.Joined())

		h.Sleep()

		quitErr := conn.Quit(ctx)
		if quitErr != birc.ErrConnectionClosed {
			assert.NilError(t, quitErr)
		}

		assert.Equal(t, io.EOF, errFromErrChan(ctx, connErr))

		h.StopServer()
		h.Wait()

		h.AssertMessages(serverMessages,
			ircx.Pass("pass"),
			ircx.Nick("nick"),
			ircx.Join("#foobar"),
			ircx.Part("#foobar"),
			ircx.Quit(),
		)

		h.AssertMessages(clientMessages)

		assert.Assert(t, conn.NumJoined() == 0)
		assert.Assert(t, conn.IsJoined("#foobar") == false)
	})
}

func TestConnectionEmptyJoinPart(t *testing.T) {
	doTest(t, func(ctx context.Context, t *testing.T, h *fakeirc.Helper, d birc.Dialer, sm <-chan *irc.Message) {
		serverMessages := h.CollectSentToServer()

		c := birc.Config{
			UserConfig: birc.UserConfig{
				Nick: "nick",
				Pass: "pass",
			},
			Dialer: &d,
		}

		connErr := make(chan error, 1)
		conn := birc.NewConnection(c)

		go func() {
			connErr <- conn.Run(ctx)
		}()

		clientMessages := h.CollectFromChannel(conn.Incoming())

		assert.NilError(t, conn.WaitUntilReady(ctx))
		h.Sleep()

		assert.NilError(t, conn.Join(ctx, ""))
		assert.NilError(t, conn.Part(ctx, ""))

		quitErr := conn.Quit(ctx)
		if quitErr != birc.ErrConnectionClosed {
			assert.NilError(t, quitErr)
		}

		assert.Equal(t, io.EOF, errFromErrChan(ctx, connErr))

		h.StopServer()
		h.Wait()

		h.AssertMessages(serverMessages,
			ircx.Pass("pass"),
			ircx.Nick("nick"),
			ircx.Quit(),
		)

		h.AssertMessages(clientMessages)
	})
}

func TestConnectionJoinTwice(t *testing.T) {
	doTest(t, func(ctx context.Context, t *testing.T, h *fakeirc.Helper, d birc.Dialer, sm <-chan *irc.Message) {
		serverMessages := h.CollectSentToServer()

		c := birc.Config{
			UserConfig: birc.UserConfig{
				Nick: "nick",
				Pass: "pass",
			},
			Dialer: &d,
		}

		connErr := make(chan error, 1)
		conn := birc.NewConnection(c)

		go func() {
			connErr <- conn.Run(ctx)
		}()

		clientMessages := h.CollectFromChannel(conn.Incoming())

		assert.NilError(t, conn.WaitUntilReady(ctx))
		h.Sleep()

		assert.Assert(t, conn.NumJoined() == 0)
		assert.Assert(t, !conn.IsJoined("#foobar"))
		assert.DeepEqual(t, []string{}, conn.Joined())

		assert.NilError(t, conn.Join(ctx, "#foobar"))
		assert.Assert(t, conn.NumJoined() == 1)
		assert.Assert(t, conn.IsJoined("#foobar"))
		assert.DeepEqual(t, []string{"#foobar"}, conn.Joined())

		assert.NilError(t, conn.Join(ctx, "#foobar"))
		assert.Assert(t, conn.NumJoined() == 1)
		assert.Assert(t, conn.IsJoined("#foobar"))
		assert.DeepEqual(t, []string{"#foobar"}, conn.Joined())

		quitErr := conn.Quit(ctx)
		if quitErr != birc.ErrConnectionClosed {
			assert.NilError(t, quitErr)
		}

		assert.Equal(t, io.EOF, errFromErrChan(ctx, connErr))

		h.StopServer()
		h.Wait()

		h.AssertMessages(serverMessages,
			ircx.Pass("pass"),
			ircx.Nick("nick"),
			ircx.Join("#foobar"),
			ircx.Quit(),
		)

		h.AssertMessages(clientMessages)

		assert.Assert(t, conn.NumJoined() == 1)
		assert.Assert(t, conn.IsJoined("#foobar") == true)
	})
}

func TestConnectionIsJoinedEmpty(t *testing.T) {
	c := birc.NewConnection(birc.Config{})
	assert.Assert(t, c.IsJoined("") == false)
}

func TestConnectionUnjoined(t *testing.T) {
	c := birc.NewConnection(birc.Config{})
	assert.Assert(t, c.IsJoined("#foobar") == false)
	assert.Assert(t, c.NumJoined() == 0)
}

func TestConnectionInitialChannels(t *testing.T) {
	doTest(t, func(ctx context.Context, t *testing.T, h *fakeirc.Helper, d birc.Dialer, sm <-chan *irc.Message) {
		serverMessages := h.CollectSentToServer()

		c := birc.Config{
			UserConfig: birc.UserConfig{
				Nick: "nick",
				Pass: "pass",
			},
			Dialer:          &d,
			InitialChannels: []string{"", "#foobar", ""},
		}

		connErr := make(chan error, 1)
		conn := birc.NewConnection(c)

		go func() {
			connErr <- conn.Run(ctx)
		}()

		clientMessages := h.CollectFromChannel(conn.Incoming())

		assert.NilError(t, conn.WaitUntilReady(ctx))
		h.Sleep()

		quitErr := conn.Quit(ctx)
		if quitErr != birc.ErrConnectionClosed {
			assert.NilError(t, quitErr)
		}

		assert.Equal(t, io.EOF, errFromErrChan(ctx, connErr))

		h.StopServer()
		h.Wait()

		h.AssertMessages(serverMessages,
			ircx.Pass("pass"),
			ircx.Nick("nick"),
			ircx.Join("#foobar"),
			ircx.Quit(),
		)

		h.AssertMessages(clientMessages)

		assert.Assert(t, conn.NumJoined() == 1)
		assert.Assert(t, conn.IsJoined("#foobar") == true)
	})
}

func TestConnectionCapabilities(t *testing.T) {
	doTest(t, func(ctx context.Context, t *testing.T, h *fakeirc.Helper, d birc.Dialer, sm <-chan *irc.Message) {
		serverMessages := h.CollectSentToServer()

		c := birc.Config{
			UserConfig: birc.UserConfig{
				Nick: "nick",
				Pass: "pass",
			},
			Dialer: &d,
			Caps:   []string{"my.cool/cap"},
		}

		connErr := make(chan error, 1)
		conn := birc.NewConnection(c)

		go func() {
			connErr <- conn.Run(ctx)
		}()

		clientMessages := h.CollectFromChannel(conn.Incoming())

		assert.NilError(t, conn.WaitUntilReady(ctx))
		h.Sleep()

		quitErr := conn.Quit(ctx)
		if quitErr != birc.ErrConnectionClosed {
			assert.NilError(t, quitErr)
		}

		assert.Equal(t, io.EOF, errFromErrChan(ctx, connErr))

		h.StopServer()
		h.Wait()

		h.AssertMessages(serverMessages,
			ircx.Pass("pass"),
			ircx.Nick("nick"),
			ircx.CapReq("my.cool/cap"),
			ircx.Quit(),
		)

		h.AssertMessages(clientMessages)
	})
}

func TestConnectionCloseAfterFirst(t *testing.T) {
	doTest(t, func(ctx context.Context, t *testing.T, h *fakeirc.Helper, d birc.Dialer, sm <-chan *irc.Message) {
		stopChan := make(chan error, 1)

		go func() {
			for {
				select {
				case _, ok := <-sm:
					if !ok {
						stopChan <- fmt.Errorf("channel closed without messages")
						return
					}

					stopChan <- h.StopServerErr()
					return

				case <-ctx.Done():
					stopChan <- ctx.Err()
					return
				}
			}
		}()

		c := birc.Config{
			UserConfig: birc.UserConfig{
				Nick: "nick",
				Pass: "pass",
			},
			Dialer: &d,
		}

		connErr := make(chan error, 1)
		conn := birc.NewConnection(c)

		go func() {
			connErr <- conn.Run(ctx)
		}()

		clientMessages := h.CollectFromChannel(conn.Incoming())

		connErrV := errFromErrChan(ctx, connErr)
		if connErrV != io.EOF {
			assert.ErrorContains(t, connErrV, "connection reset")
		}

		assert.NilError(t, errFromErrChan(ctx, stopChan))

		h.StopServer()
		h.Wait()

		h.AssertMessages(clientMessages)
	})
}

func TestConnectionReconnect(t *testing.T) {
	doTest(t, func(ctx context.Context, t *testing.T, h *fakeirc.Helper, d birc.Dialer, sm <-chan *irc.Message) {
		serverMessages := h.CollectSentToServer()

		c := birc.Config{
			UserConfig: birc.UserConfig{
				Nick: "nick",
				Pass: "pass",
			},
			Dialer: &d,
		}

		connErr := make(chan error, 1)
		conn := birc.NewConnection(c)

		go func() {
			connErr <- conn.Run(ctx)
		}()

		clientMessages := h.CollectFromChannel(conn.Incoming())

		assert.NilError(t, conn.WaitUntilReady(ctx))
		h.Sleep()

		reconn := &irc.Message{Command: "RECONNECT"}

		h.SendAsServer(ctx, reconn)

		assert.Equal(t, birc.ErrReconnect, errFromErrChan(ctx, connErr))

		h.StopServer()
		h.Wait()

		h.AssertMessages(serverMessages,
			ircx.Pass("pass"),
			ircx.Nick("nick"),
		)

		h.AssertMessages(clientMessages, reconn)
	})
}

func TestConnectionReadOnly(t *testing.T) {
	doTest(t, func(ctx context.Context, t *testing.T, h *fakeirc.Helper, d birc.Dialer, sm <-chan *irc.Message) {
		serverMessages := h.CollectSentToServer()

		c := birc.Config{
			UserConfig: birc.UserConfig{
				Nick:     "nick",
				Pass:     "pass",
				ReadOnly: true,
			},
			Dialer: &d,
		}

		connErr := make(chan error, 1)
		conn := birc.NewConnection(c)

		go func() {
			connErr <- conn.Run(ctx)
		}()

		clientMessages := h.CollectFromChannel(conn.Incoming())

		assert.NilError(t, conn.WaitUntilReady(ctx))
		h.Sleep()

		assert.Equal(t, birc.ErrReadOnly, conn.SendMessage(ctx, "#foobar", "test"))

		quitErr := conn.Quit(ctx)
		if quitErr != birc.ErrConnectionClosed {
			assert.NilError(t, quitErr)
		}

		assert.Equal(t, io.EOF, errFromErrChan(ctx, connErr))
		h.StopServer()
		h.Wait()

		h.AssertMessages(serverMessages,
			ircx.Pass("pass"),
			ircx.Nick("nick"),
			ircx.Quit(),
		)

		h.AssertMessages(clientMessages)
	})
}

func TestConnectionPing(t *testing.T) {
	doTestSecureInsecure(t, func(ctx context.Context, t *testing.T, h *fakeirc.Helper, d birc.Dialer, sm <-chan *irc.Message) {
		serverMessages := h.CollectSentToServer()

		const dur = time.Second / 2

		c := birc.Config{
			UserConfig: birc.UserConfig{
				Nick: "nick",
				Pass: "pass",
			},
			Dialer:       &d,
			PingInterval: dur,
			PingDeadline: dur / 2,
		}

		connErr := make(chan error, 1)
		conn := birc.NewConnection(c)

		go func() {
			connErr <- conn.Run(ctx)
		}()

		_ = h.CollectFromChannel(conn.Incoming())

		assert.NilError(t, conn.WaitUntilReady(ctx))
		h.Sleep()

		time.Sleep(3 * dur / 2)

		h.Sleep()

		quitErr := conn.Quit(ctx)
		if quitErr != birc.ErrConnectionClosed {
			assert.NilError(t, quitErr)
		}

		assert.Equal(t, io.EOF, errFromErrChan(ctx, connErr))

		h.StopServer()
		h.Wait()

		h.AssertMessages(serverMessages,
			ircx.Pass("pass"),
			ircx.Nick("nick"),
			ircx.Quit(),
		)
	})
}

func TestConnectionPingNoPong(t *testing.T) {
	doTestSecureInsecure(t, func(ctx context.Context, t *testing.T, h *fakeirc.Helper, d birc.Dialer, sm <-chan *irc.Message) {
		serverMessages := h.CollectSentToServer()

		const dur = time.Second / 2

		c := birc.Config{
			UserConfig: birc.UserConfig{
				Nick: "nick",
				Pass: "pass",
			},
			Dialer:       &d,
			PingInterval: dur,
			PingDeadline: dur / 2,
		}

		connErr := make(chan error, 1)
		conn := birc.NewConnection(c)

		go func() {
			connErr <- conn.Run(ctx)
		}()

		_ = h.CollectFromChannel(conn.Incoming())

		assert.NilError(t, conn.WaitUntilReady(ctx))
		h.Sleep()

		time.Sleep(3 * dur / 2)

		h.Sleep()

		assert.Equal(t, birc.ErrFailedPing, errFromErrChan(ctx, connErr))

		h.StopServer()
		h.Wait()

		h.AssertMessages(serverMessages,
			ircx.Pass("pass"),
			ircx.Nick("nick"),
		)
	}, fakeirc.Pong(false))
}

func TestConnectionWaitCancel(t *testing.T) {
	c := birc.NewConnection(birc.Config{})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	assert.Equal(t, c.WaitUntilReady(ctx), context.Canceled)
}
