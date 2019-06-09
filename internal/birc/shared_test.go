package birc_test

import (
	"context"
	"testing"
	"time"

	"github.com/fortytw2/leaktest"
	"github.com/hortbot/hortbot/internal/birc"
	"github.com/hortbot/hortbot/internal/pkg/ctxlog"
	"github.com/hortbot/hortbot/internal/pkg/fakeirc"
	"github.com/hortbot/hortbot/internal/pkg/testutil"
	"github.com/jakebailey/irc"
	"github.com/pkg/errors"
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
	Ping(ctx context.Context, target string) error
}

var _ connCommon = (*birc.Connection)(nil)
var _ connCommon = (*birc.Pool)(nil)

func doTestSecureInsecure(
	t *testing.T,
	fn func(ctx context.Context, t *testing.T, h *fakeirc.Helper, d birc.Dialer, sm <-chan *irc.Message),
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
		test := test
		t.Run(test.name, func(t *testing.T) {
			doTestHelper(t, test.insecure, fn)
		})
	}
}

func doTest(
	t *testing.T,
	fn func(ctx context.Context, t *testing.T, h *fakeirc.Helper, d birc.Dialer, sm <-chan *irc.Message),
) {
	t.Helper()
	doTestHelper(t, true, fn)
}

func doTestHelper(
	t *testing.T,
	insecure bool,
	fn func(ctx context.Context, t *testing.T, h *fakeirc.Helper, d birc.Dialer, sm <-chan *irc.Message),
) {
	t.Helper()

	if testing.Short() {
		t.Skip("skipping test in short mode")
		return
	}

	defer leaktest.Check(t)()

	var opts []fakeirc.Option

	if !insecure {
		opts = []fakeirc.Option{fakeirc.TLS(fakeirc.TLSConfig)}
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
