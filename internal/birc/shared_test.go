package birc_test

import (
	"context"
	"testing"
	"time"

	"github.com/fortytw2/leaktest"
	"github.com/friendsofgo/errors"
	"github.com/hortbot/hortbot/internal/birc"
	"github.com/hortbot/hortbot/internal/birc/fakeirc"
	"github.com/hortbot/hortbot/internal/pkg/testutil"
	"github.com/jakebailey/irc"
	"github.com/zikaeroh/ctxlog"
)

func testContext() (context.Context, context.CancelFunc) {
	ctx := context.Background()
	return context.WithTimeout(ctx, 5*time.Second)
}

func canceledContext(ctx context.Context) context.Context {
	ctx, cancel := context.WithCancel(ctx)
	cancel()
	return ctx
}

type connCommon interface {
	SendMessage(ctx context.Context, target string, message string) error
	Quit(ctx context.Context) error
	Incoming() <-chan *irc.Message
	Join(ctx context.Context, channels ...string) error
	Part(ctx context.Context, channels ...string) error
}

var (
	_ connCommon = (*birc.Connection)(nil)
	_ connCommon = (*birc.Pool)(nil)
)

func doTestSecureInsecure(
	t *testing.T,
	fn func(ctx context.Context, t *testing.T, h *fakeirc.Helper, d birc.Dialer, sm <-chan *irc.Message),
	opts ...fakeirc.Option,
) {
	t.Helper()

	tests := []struct {
		name     string
		insecure bool
	}{
		{"Insecure", true},
		{"Secure", false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			doTestHelper(t, test.insecure, fn, opts...)
		})
	}
}

func doTest(
	t *testing.T,
	fn func(ctx context.Context, t *testing.T, h *fakeirc.Helper, d birc.Dialer, sm <-chan *irc.Message),
	opts ...fakeirc.Option,
) {
	t.Helper()
	doTestHelper(t, true, fn, opts...)
}

func doTestHelper(
	t *testing.T,
	insecure bool,
	fn func(ctx context.Context, t *testing.T, h *fakeirc.Helper, d birc.Dialer, sm <-chan *irc.Message),
	opts ...fakeirc.Option,
) {
	t.Helper()

	defer leaktest.Check(t)() // Must not be parallel.

	if !insecure {
		opts = append(opts, fakeirc.TLS(fakeirc.TLSConfig))
	}

	ctx, cancel := testContext()
	defer cancel()

	ctx = ctxlog.WithLogger(ctx, testutil.Logger(t))

	h := fakeirc.NewHelper(ctx, t, opts...)
	defer h.StopServer()

	d := birc.Dialer{
		Addr:     h.Addr(),
		Insecure: insecure,
	}

	if !insecure {
		d.TLSConfig = fakeirc.TLSConfig
	}

	fn(ctx, t, h, d, h.ServerMessages())
}

func errFromErrChan(ctx context.Context, errChan chan error) error {
	select {
	case err := <-errChan:
		return err
	case <-ctx.Done():
		return errors.WithMessage(ctx.Err(), "errFromErrChan cancel")
	}
}
