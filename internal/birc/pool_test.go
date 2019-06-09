package birc_test

import (
	"context"
	"testing"
	"time"

	"github.com/hortbot/hortbot/internal/birc"
	"github.com/hortbot/hortbot/internal/pkg/fakeirc"
	"github.com/hortbot/hortbot/internal/pkg/ircx"
	"github.com/jakebailey/irc"
	"gotest.tools/assert"
)

func TestPoolUnused(t *testing.T) {
	p := birc.NewPool(birc.PoolConfig{})
	assert.Assert(t, p != nil)
	assert.Assert(t, !p.IsJoined(""))
	p.Stop()
}

func TestPoolRunStop(t *testing.T) {
	doTest(t, func(ctx context.Context, t *testing.T, h *fakeirc.Helper, d birc.Dialer, sm <-chan *irc.Message) {
		serverMessages := h.CollectFromServer()

		c := birc.PoolConfig{
			Config: birc.Config{
				UserConfig: birc.UserConfig{
					Nick: "nick",
					Pass: "pass",
				},
				Dialer: &d,
			},
			JoinRate: 100,
		}

		errChan := make(chan error, 1)
		pool := birc.NewPool(c)
		defer pool.Stop()

		go func() {
			errChan <- pool.Run(ctx)
		}()

		clientMessages := h.CollectFromChannel(pool.Incoming())

		assert.NilError(t, pool.WaitUntilReady(ctx))
		h.Sleep()

		// TODO: Make WaitUntilReady wait for the initial connection to occur.
		h.Sleep()
		h.Sleep()
		assert.Assert(t, pool.NumConns() == 1)

		pool.Stop()

		assert.Equal(t, birc.ErrPoolStopped, errFromErrChan(ctx, errChan))

		h.StopServer()
		h.Wait()

		h.AssertMessages(serverMessages,
			ircx.Pass("pass"),
			ircx.Nick("nick"),
		)
		h.AssertMessages(clientMessages)

		assert.Assert(t, pool.NumJoined() == 0)
		assert.Assert(t, pool.IsJoined("#foobar") == false)
		assert.DeepEqual(t, []string{}, pool.Joined())
	})
}

func TestPoolJoinOne(t *testing.T) {
	doTest(t, func(ctx context.Context, t *testing.T, h *fakeirc.Helper, d birc.Dialer, sm <-chan *irc.Message) {
		serverMessages := h.CollectFromServer()

		c := birc.PoolConfig{
			Config: birc.Config{
				UserConfig: birc.UserConfig{
					Nick: "nick",
					Pass: "pass",
				},
				Dialer: &d,
			},
			JoinRate: 100,
		}

		errChan := make(chan error, 1)
		pool := birc.NewPool(c)
		defer pool.Stop()

		go func() {
			errChan <- pool.Run(ctx)
		}()

		clientMessages := h.CollectFromChannel(pool.Incoming())

		assert.NilError(t, pool.WaitUntilReady(ctx))
		h.Sleep()

		assert.Equal(t, pool.NumJoined(), 0)
		assert.Assert(t, !pool.IsJoined("#foobar"))
		assert.DeepEqual(t, []string{}, pool.Joined())

		assert.NilError(t, pool.Join(ctx, "#foobar"))
		assert.Equal(t, pool.NumConns(), 1)
		assert.Equal(t, pool.NumJoined(), 1)
		assert.Assert(t, pool.IsJoined("#foobar"))
		assert.DeepEqual(t, []string{"#foobar"}, pool.Joined())

		h.Sleep()

		pool.Stop()

		assert.Equal(t, birc.ErrPoolStopped, errFromErrChan(ctx, errChan))

		h.StopServer()
		h.Wait()

		h.AssertMessages(serverMessages,
			ircx.Pass("pass"),
			ircx.Nick("nick"),
			ircx.Join("#foobar"),
		)

		h.AssertMessages(clientMessages)
	})
}

func TestPoolChannelMessage(t *testing.T) {
	doTest(t, func(ctx context.Context, t *testing.T, h *fakeirc.Helper, d birc.Dialer, sm <-chan *irc.Message) {
		serverMessages := h.CollectFromServer()

		c := birc.PoolConfig{
			Config: birc.Config{
				UserConfig: birc.UserConfig{
					Nick: "nick",
					Pass: "pass",
				},
				Dialer: &d,
			},
			JoinRate: -1,
		}

		errChan := make(chan error, 1)
		pool := birc.NewPool(c)
		defer pool.Stop()

		go func() {
			errChan <- pool.Run(ctx)
		}()

		clientMessages := h.CollectFromChannel(pool.Incoming())

		assert.NilError(t, pool.WaitUntilReady(ctx))
		h.Sleep()

		assert.NilError(t, pool.Join(ctx, "#foobar"))
		assert.NilError(t, pool.Join(ctx, "#barfoo"))

		h.Sleep()
		h.Sleep()

		m := ircx.PrivMsg("#foobar", "test1")
		h.SendToServer(ctx, m)
		h.Sleep()

		assert.NilError(t, pool.Part(ctx, "#foobar"))
		h.Sleep()
		h.Sleep()

		h.SendToServer(ctx, m)
		h.Sleep()

		pool.Stop()

		assert.Equal(t, birc.ErrPoolStopped, errFromErrChan(ctx, errChan))

		h.StopServer()
		h.Wait()

		h.AssertMessages(serverMessages,
			ircx.Pass("pass"),
			ircx.Nick("nick"),
			ircx.Join("#foobar"),
			ircx.Join("#barfoo"),
			ircx.Part("#foobar"),
		)

		h.AssertMessages(clientMessages, m)
	})
}

func TestPoolSyncJoined(t *testing.T) {
	doTest(t, func(ctx context.Context, t *testing.T, h *fakeirc.Helper, d birc.Dialer, sm <-chan *irc.Message) {
		serverMessages := h.CollectFromServer()

		c := birc.PoolConfig{
			Config: birc.Config{
				UserConfig: birc.UserConfig{
					Nick: "nick",
					Pass: "pass",
				},
				Dialer: &d,
			},
			JoinRate: 1000,
		}

		errChan := make(chan error, 1)
		pool := birc.NewPool(c)
		defer pool.Stop()

		go func() {
			errChan <- pool.Run(ctx)
		}()

		clientMessages := h.CollectFromChannel(pool.Incoming())

		assert.NilError(t, pool.WaitUntilReady(ctx))
		h.Sleep()

		assert.NilError(t, pool.SyncJoined(ctx, "#foobar", "#barfoo"))
		assert.Equal(t, pool.NumJoined(), 2)
		assert.Assert(t, pool.IsJoined("#foobar"))
		assert.Assert(t, pool.IsJoined("#barfoo"))
		assert.DeepEqual(t, []string{"#barfoo", "#foobar"}, pool.Joined())

		m := ircx.PrivMsg("#foobar", "test1")
		h.SendToServer(ctx, m)
		h.Sleep()
		h.Sleep()

		assert.NilError(t, pool.SyncJoined(ctx, "#barfoo"))
		assert.Equal(t, pool.NumJoined(), 1)
		assert.Assert(t, !pool.IsJoined("#foobar"))
		assert.Assert(t, pool.IsJoined("#barfoo"))
		assert.DeepEqual(t, []string{"#barfoo"}, pool.Joined())

		h.Sleep()
		h.Sleep()
		h.SendToServer(ctx, m)

		pool.Prune()
		h.Sleep()

		assert.Equal(t, pool.NumConns(), 1)

		pool.Stop()

		assert.Equal(t, birc.ErrPoolStopped, errFromErrChan(ctx, errChan))

		h.StopServer()
		h.Wait()

		h.AssertMessages(serverMessages,
			ircx.Pass("pass"),
			ircx.Nick("nick"),
			ircx.Join("#barfoo"),
			ircx.Join("#foobar"),
			ircx.Part("#foobar"),
		)

		h.AssertMessages(clientMessages, m)
	})
}

func TestPoolSendMessage(t *testing.T) {
	doTest(t, func(ctx context.Context, t *testing.T, h *fakeirc.Helper, d birc.Dialer, sm <-chan *irc.Message) {
		serverMessages := h.CollectFromServer()

		c := birc.PoolConfig{
			Config: birc.Config{
				UserConfig: birc.UserConfig{
					Nick: "nick",
					Pass: "pass",
				},
				Dialer: &d,
			},
			JoinRate: -1,
		}

		errChan := make(chan error, 1)
		pool := birc.NewPool(c)
		defer pool.Stop()

		go func() {
			errChan <- pool.Run(ctx)
		}()

		clientMessages := h.CollectFromChannel(pool.Incoming())

		assert.NilError(t, pool.WaitUntilReady(ctx))
		h.Sleep()

		assert.NilError(t, pool.Join(ctx, "#foobar"))
		assert.NilError(t, pool.SendMessage(ctx, "#foobar", "test"))

		h.Sleep()

		pool.Stop()

		assert.Equal(t, birc.ErrPoolStopped, errFromErrChan(ctx, errChan))

		h.StopServer()
		h.Wait()

		h.AssertMessages(serverMessages,
			ircx.Pass("pass"),
			ircx.Nick("nick"),
			ircx.Join("#foobar"),
			ircx.PrivMsg("#foobar", "test"),
		)

		h.AssertMessages(clientMessages)
	})
}

func TestPoolPrune(t *testing.T) {
	doTest(t, func(ctx context.Context, t *testing.T, h *fakeirc.Helper, d birc.Dialer, sm <-chan *irc.Message) {
		serverMessages := h.CollectFromServer()

		c := birc.PoolConfig{
			Config: birc.Config{
				UserConfig: birc.UserConfig{
					Nick: "nick",
					Pass: "pass",
				},
				Dialer: &d,
			},
			MaxChannelsPerSubConn: 1,
			JoinRate:              -1,
		}

		errChan := make(chan error, 1)
		pool := birc.NewPool(c)
		defer pool.Stop()

		go func() {
			errChan <- pool.Run(ctx)
		}()

		clientMessages := h.CollectFromChannel(pool.Incoming())

		assert.NilError(t, pool.WaitUntilReady(ctx))
		h.Sleep()

		assert.NilError(t, pool.Join(ctx, "#foobar"))
		h.Sleep()
		assert.NilError(t, pool.Join(ctx, "#barfoo"))
		assert.Equal(t, pool.NumConns(), 2)
		assert.Equal(t, pool.NumJoined(), 2)
		assert.Assert(t, pool.IsJoined("#foobar"))
		assert.Assert(t, pool.IsJoined("#barfoo"))
		assert.DeepEqual(t, []string{"#barfoo", "#foobar"}, pool.Joined())

		h.Sleep()

		assert.NilError(t, pool.Part(ctx, "#foobar"))
		assert.Equal(t, pool.NumJoined(), 1)
		assert.Assert(t, !pool.IsJoined("#foobar"))
		assert.Assert(t, pool.IsJoined("#barfoo"))
		assert.DeepEqual(t, []string{"#barfoo"}, pool.Joined())

		pool.Prune()
		h.Sleep()

		assert.Equal(t, pool.NumConns(), 1)

		pool.Prune()
		h.Sleep()

		assert.Equal(t, pool.NumConns(), 1)

		pool.Stop()

		assert.Equal(t, birc.ErrPoolStopped, errFromErrChan(ctx, errChan))

		h.StopServer()
		h.Wait()

		h.AssertMessages(serverMessages,
			ircx.Pass("pass"),
			ircx.Nick("nick"),
			ircx.Join("#foobar"),
			ircx.Pass("pass"),
			ircx.Nick("nick"),
			ircx.Join("#barfoo"),
			ircx.Part("#foobar"),
		)

		h.AssertMessages(clientMessages)
	})
}

func TestPoolPruneAuto(t *testing.T) {
	doTest(t, func(ctx context.Context, t *testing.T, h *fakeirc.Helper, d birc.Dialer, sm <-chan *irc.Message) {
		serverMessages := h.CollectFromServer()

		pruneInterval := h.SleepDur * 5

		c := birc.PoolConfig{
			Config: birc.Config{
				UserConfig: birc.UserConfig{
					Nick: "nick",
					Pass: "pass",
				},
				Dialer: &d,
			},
			PruneInterval:         pruneInterval,
			MaxChannelsPerSubConn: 1,
			JoinRate:              -1,
		}

		errChan := make(chan error, 1)
		pool := birc.NewPool(c)
		defer pool.Stop()

		pruned := time.After(pruneInterval)

		go func() {
			errChan <- pool.Run(ctx)
		}()

		clientMessages := h.CollectFromChannel(pool.Incoming())

		assert.NilError(t, pool.WaitUntilReady(ctx))
		h.Sleep()

		assert.NilError(t, pool.Join(ctx, "#foobar"))
		h.Sleep()

		assert.NilError(t, pool.Join(ctx, "#barfoo"))
		h.Sleep()

		assert.Equal(t, pool.NumConns(), 2)
		assert.Equal(t, pool.NumJoined(), 2)
		assert.Assert(t, pool.IsJoined("#foobar"))
		assert.Assert(t, pool.IsJoined("#barfoo"))
		assert.DeepEqual(t, []string{"#barfoo", "#foobar"}, pool.Joined())

		h.Sleep()

		select {
		case <-pruned:
			t.Fatal("prune happened before second part could occur")
		default:
		}

		assert.NilError(t, pool.Part(ctx, "#foobar"))
		assert.Equal(t, pool.NumJoined(), 1)
		assert.Assert(t, !pool.IsJoined("#foobar"))
		assert.Assert(t, pool.IsJoined("#barfoo"))
		assert.DeepEqual(t, []string{"#barfoo"}, pool.Joined())

		select {
		case <-pruned:
		case <-ctx.Done():
			assert.NilError(t, ctx.Err())
		}

		h.Sleep()

		assert.Equal(t, pool.NumConns(), 1)

		pool.Stop()

		assert.Equal(t, birc.ErrPoolStopped, errFromErrChan(ctx, errChan))

		h.StopServer()
		h.Wait()

		h.AssertMessages(serverMessages,
			ircx.Pass("pass"),
			ircx.Nick("nick"),
			ircx.Join("#foobar"),
			ircx.Pass("pass"),
			ircx.Nick("nick"),
			ircx.Join("#barfoo"),
			ircx.Part("#foobar"),
		)

		h.AssertMessages(clientMessages)
	})
}

func TestPoolQuitRejoin(t *testing.T) {
	doTest(t, func(ctx context.Context, t *testing.T, h *fakeirc.Helper, d birc.Dialer, sm <-chan *irc.Message) {
		serverMessages := h.CollectFromServer()

		c := birc.PoolConfig{
			Config: birc.Config{
				UserConfig: birc.UserConfig{
					Nick: "nick",
					Pass: "pass",
				},
				Dialer: &d,
			},
			JoinRate: -1,
		}

		errChan := make(chan error, 1)
		pool := birc.NewPool(c)
		defer pool.Stop()

		go func() {
			errChan <- pool.Run(ctx)
		}()

		clientMessages := h.CollectFromChannel(pool.Incoming())

		assert.NilError(t, pool.WaitUntilReady(ctx))
		h.Sleep()

		assert.NilError(t, pool.Join(ctx, "#foobar"))

		m := ircx.PrivMsg("#foobar", "test1")
		h.Sleep()
		h.SendToServer(ctx, m)

		assert.NilError(t, pool.Quit(ctx))

		h.Sleep()
		h.Sleep()

		pool.Stop()

		assert.Equal(t, birc.ErrPoolStopped, errFromErrChan(ctx, errChan))

		h.StopServer()
		h.Wait()

		h.AssertMessages(serverMessages,
			ircx.Pass("pass"),
			ircx.Nick("nick"),
			ircx.Join("#foobar"),
			ircx.Quit(),
			ircx.Pass("pass"),
			ircx.Nick("nick"),
			ircx.Join("#foobar"),
		)

		h.AssertMessages(clientMessages, m)
	})
}

func TestPoolPing(t *testing.T) {
	doTest(t, func(ctx context.Context, t *testing.T, h *fakeirc.Helper, d birc.Dialer, sm <-chan *irc.Message) {
		serverMessages := h.CollectFromServer()

		c := birc.PoolConfig{
			Config: birc.Config{
				UserConfig: birc.UserConfig{
					Nick: "nick",
					Pass: "pass",
				},
				Dialer: &d,
			},
			JoinRate: -1,
		}

		errChan := make(chan error, 1)
		pool := birc.NewPool(c)
		defer pool.Stop()

		go func() {
			errChan <- pool.Run(ctx)
		}()

		clientMessages := h.CollectFromChannel(pool.Incoming())

		assert.NilError(t, pool.WaitUntilReady(ctx))
		h.Sleep()

		assert.NilError(t, pool.Join(ctx, "#foobar"))
		assert.NilError(t, pool.Ping(ctx, "test"))

		h.Sleep()

		pool.Stop()

		assert.Equal(t, birc.ErrPoolStopped, errFromErrChan(ctx, errChan))

		h.StopServer()
		h.Wait()

		h.AssertMessages(serverMessages,
			ircx.Pass("pass"),
			ircx.Nick("nick"),
			ircx.Join("#foobar"),
			&irc.Message{Command: "PING", Trailing: "test"},
		)

		h.AssertMessages(clientMessages)
	})
}

func TestPoolNotJoinedSend(t *testing.T) {
	doTest(t, func(ctx context.Context, t *testing.T, h *fakeirc.Helper, d birc.Dialer, sm <-chan *irc.Message) {
		serverMessages := h.CollectFromServer()

		c := birc.PoolConfig{
			Config: birc.Config{
				UserConfig: birc.UserConfig{
					Nick: "nick",
					Pass: "pass",
				},
				Dialer: &d,
			},
			JoinRate: -1,
		}

		errChan := make(chan error, 1)
		pool := birc.NewPool(c)
		defer pool.Stop()

		go func() {
			errChan <- pool.Run(ctx)
		}()

		clientMessages := h.CollectFromChannel(pool.Incoming())

		assert.NilError(t, pool.WaitUntilReady(ctx))
		h.Sleep()

		assert.NilError(t, pool.SendMessage(ctx, "#foobar", "test"))
		h.Sleep()

		pool.Stop()

		assert.Equal(t, birc.ErrPoolStopped, errFromErrChan(ctx, errChan))

		h.StopServer()
		h.Wait()

		h.AssertMessages(serverMessages,
			ircx.Pass("pass"),
			ircx.Nick("nick"),
			ircx.PrivMsg("#foobar", "test"),
		)

		h.AssertMessages(clientMessages)
	})
}
