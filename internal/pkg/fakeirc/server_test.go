package fakeirc_test

import (
	"context"
	"testing"
	"time"

	"github.com/fortytw2/leaktest"
	"github.com/hortbot/hortbot/internal/pkg/fakeirc"
	"github.com/hortbot/hortbot/internal/pkg/ircx"
	"github.com/jakebailey/irc"
	"gotest.tools/v3/assert"
)

func TestServerUnused(t *testing.T) {
	defer leaktest.Check(t)()

	ctx, cancel := testContext()
	defer cancel()
	h := fakeirc.NewHelper(ctx, t)
	assert.Assert(t, h.Addr() != "")

	defer h.StopServer()

	serverMessages := h.CollectFromServer()

	h.StopServer()

	h.Wait()
	h.AssertMessages(serverMessages)

	err := h.SendToServerErr(ctx, &irc.Message{
		Command: "WOW",
	})
	assert.Equal(t, err, fakeirc.ErrStopped)
}

func TestServerNoMessages(t *testing.T) {
	defer leaktest.Check(t)()

	ctx, cancel := testContext()
	defer cancel()
	h := fakeirc.NewHelper(ctx, t)

	defer h.StopServer()
	serverMessages := h.CollectFromServer()

	conn := h.Dial()
	defer h.CloseConn(conn)
	clientMessages := h.CollectFromConn(conn)

	h.CloseConn(conn)
	h.StopServer()

	h.Wait()
	h.AssertMessages(serverMessages)
	h.AssertMessages(clientMessages)
}

func TestServerBroadcast(t *testing.T) {
	t.Skip("flaky")

	defer leaktest.Check(t)()

	ctx, cancel := testContext()
	defer cancel()
	h := fakeirc.NewHelper(ctx, t)

	defer h.StopServer()
	serverMessages := h.CollectFromServer()

	conn := h.Dial()
	defer h.CloseConn(conn)
	clientMessages := h.CollectFromConn(conn)

	m := &irc.Message{
		Command: "WOW",
	}

	h.SendToServer(ctx, m)

	// This is flaky; not sure why.
	h.Sleep()
	h.Sleep()
	h.Sleep()
	h.Sleep()
	h.Sleep()

	h.CloseConn(conn)
	h.StopServer()

	h.Wait()
	h.AssertMessages(serverMessages)
	h.AssertMessages(clientMessages, m)
}

func TestServerFilterNoJoin(t *testing.T) {
	defer leaktest.Check(t)()

	ctx, cancel := testContext()
	defer cancel()
	h := fakeirc.NewHelper(ctx, t)

	defer h.StopServer()
	serverMessages := h.CollectFromServer()

	conn := h.Dial()
	defer h.CloseConn(conn)
	clientMessages := h.CollectFromConn(conn)

	m := fakeirc.TagChannel(&irc.Message{Command: "WOW"}, "#foobar")

	h.SendToServer(ctx, m)

	h.CloseConn(conn)
	h.StopServer()

	h.Wait()
	h.AssertMessages(serverMessages)
	h.AssertMessages(clientMessages)
}

func TestServerPrivMsgNoJoin(t *testing.T) {
	defer leaktest.Check(t)()

	ctx, cancel := testContext()
	defer cancel()
	h := fakeirc.NewHelper(ctx, t)

	defer h.StopServer()
	serverMessages := h.CollectFromServer()

	conn := h.Dial()
	defer h.CloseConn(conn)
	clientMessages := h.CollectFromConn(conn)

	m := ircx.PrivMsg("#foobar", "test")

	h.SendToServer(ctx, m)

	h.CloseConn(conn)
	h.StopServer()

	h.Wait()
	h.AssertMessages(serverMessages)
	h.AssertMessages(clientMessages)
}

func TestServerSinglePrivMsg(t *testing.T) {
	defer leaktest.Check(t)()

	ctx, cancel := testContext()
	defer cancel()
	h := fakeirc.NewHelper(ctx, t)

	defer h.StopServer()
	serverMessages := h.CollectFromServer()

	conn := h.Dial()
	defer h.CloseConn(conn)
	clientMessages := h.CollectFromConn(conn)

	join := ircx.Join("#foobar")

	h.SendWithConn(conn, join)

	m := ircx.PrivMsg("#foobar", "test")

	h.SendToServer(ctx, m)

	h.CloseConn(conn)
	h.StopServer()

	h.Wait()
	h.AssertMessages(serverMessages, join)
	h.AssertMessages(clientMessages, m)
}

func TestServerSingleFiltered(t *testing.T) {
	defer leaktest.Check(t)()

	ctx, cancel := testContext()
	defer cancel()
	h := fakeirc.NewHelper(ctx, t)

	defer h.StopServer()
	serverMessages := h.CollectFromServer()

	conn := h.Dial()
	defer h.CloseConn(conn)
	clientMessages := h.CollectFromConn(conn)

	join := ircx.Join("#foobar")

	h.SendWithConn(conn, join)

	m := fakeirc.TagChannel(&irc.Message{Command: "WOW"}, "#foobar")

	h.SendToServer(ctx, m)

	h.CloseConn(conn)
	h.StopServer()

	h.Wait()
	h.AssertMessages(serverMessages, join)
	h.AssertMessages(clientMessages, m)
}

func TestServerTwo(t *testing.T) {
	defer leaktest.Check(t)()

	ctx, cancel := testContext()
	defer cancel()
	h := fakeirc.NewHelper(ctx, t)

	defer h.StopServer()
	serverMessages := h.CollectFromServer()

	first := h.Dial()
	defer h.CloseConn(first)
	firstMessages := h.CollectFromConn(first)

	second := h.Dial()
	defer h.CloseConn(second)
	secondMessages := h.CollectFromConn(second)

	join := ircx.Join("#foobar")

	h.SendWithConn(first, join)
	h.SendWithConn(second, join)

	m := ircx.PrivMsg("#foobar", "test")
	h.SendWithConn(first, m)

	h.CloseConn(first)
	h.CloseConn(second)
	h.StopServer()

	h.Wait()
	h.AssertMessages(serverMessages, join, join, m)
	h.AssertMessages(firstMessages)
	h.AssertMessages(secondMessages, m)
}

func TestServerTwoJoinPart(t *testing.T) {
	defer leaktest.Check(t)()

	ctx, cancel := testContext()
	defer cancel()
	h := fakeirc.NewHelper(ctx, t)

	defer h.StopServer()
	serverMessages := h.CollectFromServer()

	first := h.Dial()
	defer h.CloseConn(first)
	firstMessages := h.CollectFromConn(first)

	second := h.Dial()
	defer h.CloseConn(second)
	secondMessages := h.CollectFromConn(second)
	join := ircx.Join("#foobar")

	h.SendWithConn(first, join)
	h.SendWithConn(second, join)

	m := ircx.PrivMsg("#foobar", "test")
	h.SendWithConn(first, m)

	part := ircx.Part("#foobar")
	h.SendWithConn(second, part)
	h.SendWithConn(first, m)

	h.CloseConn(first)
	h.CloseConn(second)
	h.StopServer()

	h.Wait()
	h.AssertMessages(serverMessages, join, join, m, part, m)
	h.AssertMessages(firstMessages)
	h.AssertMessages(secondMessages, m)
}

func TestServerQuit(t *testing.T) {
	defer leaktest.Check(t)()

	ctx, cancel := testContext()
	defer cancel()
	h := fakeirc.NewHelper(ctx, t)

	defer h.StopServer()
	serverMessages := h.CollectFromServer()

	conn := h.Dial()
	defer h.CloseConn(conn)
	clientMessages := h.CollectFromConn(conn)

	quit := &irc.Message{
		Command: "QUIT",
	}

	h.SendWithConn(conn, quit)

	m := ircx.PrivMsg("#foobar", "test")

	h.SendWithConn(conn, m)

	h.CloseConn(conn)
	h.StopServer()

	h.Wait()
	h.AssertMessages(serverMessages, quit)
	h.AssertMessages(clientMessages)
}

func TestServerSinglePrivMsgTLS(t *testing.T) {
	defer leaktest.Check(t)()

	ctx, cancel := testContext()
	defer cancel()
	h := fakeirc.NewHelper(ctx, t, fakeirc.TLS(fakeirc.TLSConfig))

	defer h.StopServer()
	serverMessages := h.CollectFromServer()

	conn := h.Dial()
	defer h.CloseConn(conn)
	clientMessages := h.CollectFromConn(conn)

	join := ircx.Join("#foobar")

	h.SendWithConn(conn, join)

	m := ircx.PrivMsg("#foobar", "test")

	h.SendToServer(ctx, m)

	h.CloseConn(conn)
	h.StopServer()

	h.Wait()
	h.AssertMessages(serverMessages, join)
	h.AssertMessages(clientMessages, m)
}

func TestServerPong(t *testing.T) {
	defer leaktest.Check(t)()

	ctx, cancel := testContext()
	defer cancel()
	h := fakeirc.NewHelper(ctx, t, fakeirc.RecordPings(true))

	defer h.StopServer()
	serverMessages := h.CollectFromServer()

	conn := h.Dial()
	defer h.CloseConn(conn)
	clientMessages := h.CollectFromConn(conn)

	m := &irc.Message{
		Command:  "PING",
		Trailing: "some message",
	}

	h.SendWithConn(conn, m)

	h.CloseConn(conn)
	h.StopServer()

	h.Wait()
	h.AssertMessages(serverMessages, m)
	h.AssertMessages(clientMessages, &irc.Message{Command: "PONG", Trailing: m.Trailing})
}

func TestServerNoPong(t *testing.T) {
	defer leaktest.Check(t)()

	ctx, cancel := testContext()
	defer cancel()
	h := fakeirc.NewHelper(ctx, t, fakeirc.Pong(false), fakeirc.RecordPings(true))

	defer h.StopServer()
	serverMessages := h.CollectFromServer()

	conn := h.Dial()
	defer h.CloseConn(conn)
	clientMessages := h.CollectFromConn(conn)

	m := &irc.Message{
		Command:  "PING",
		Trailing: "some message",
	}

	h.SendWithConn(conn, m)

	h.CloseConn(conn)
	h.StopServer()

	h.Wait()
	h.AssertMessages(serverMessages, m)
	h.AssertMessages(clientMessages)
}

func testContext() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), 5*time.Second)
}
