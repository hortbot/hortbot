package birc_test

import (
	"crypto/tls"
	"testing"

	"github.com/fortytw2/leaktest"
	"github.com/hortbot/hortbot/internal/birc"
	"github.com/hortbot/hortbot/internal/birc/fakeirc"
	"gotest.tools/v3/assert"
)

func TestDialerCanceled(t *testing.T) {
	defer leaktest.Check(t)()

	ctx, cancel := testContext()
	defer cancel()

	h := fakeirc.NewHelper(ctx, t)
	defer h.StopServer()
	serverMessages := h.CollectSentToServer()

	d := birc.Dialer{
		Addr:     h.Addr(),
		Insecure: true,
	}

	conn, err := d.Dial(canceledContext(ctx))
	assert.ErrorContains(t, err, "operation was canceled")
	assert.Assert(t, conn == nil)

	h.StopServer()

	h.AssertMessages(serverMessages)
}

func TestDialerBadUpgrade(t *testing.T) {
	defer leaktest.Check(t)()

	ctx, cancel := testContext()
	defer cancel()

	tlsConfig := fakeirc.TLSConfig.Clone()
	tlsConfig.ClientCAs = nil

	// In TLS 1.3, errors are propogated on the first client read.
	// This test checks a code path which can be triggered with such
	// an error, so set it to 1.2 to test the behavior.
	tlsConfig.MaxVersion = tls.VersionTLS12

	h := fakeirc.NewHelper(ctx, t, fakeirc.TLS(tlsConfig))
	defer h.StopServer()
	serverMessages := h.CollectSentToServer()

	d := birc.Dialer{
		Addr:      h.Addr(),
		TLSConfig: fakeirc.TLSConfig,
	}

	_, err := d.Dial(ctx)
	assert.ErrorContains(t, err, "bad certificate")

	assert.ErrorContains(t, h.StopServerErr(), "certificate signed by unknown authority")
	h.AssertMessages(serverMessages)
}
