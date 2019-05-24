package birc_test

import (
	"crypto/tls"
	"testing"

	"github.com/fortytw2/leaktest"
	"github.com/hortbot/hortbot/internal/birc"
	"github.com/hortbot/hortbot/internal/fakeirc"
	"gotest.tools/assert"
)

func TestDialerCanceled(t *testing.T) {
	defer leaktest.Check(t)()

	ctx, cancel := testContext()
	defer cancel()

	h := fakeirc.NewHelper(ctx, t)
	defer h.StopServer()
	serverMessages := h.CollectFromServer()

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

	// This test doesn't work in TLS 1.3 because when Dialing, no data
	// is read back from the server. Since with this IRC client PASS/NICK
	// get sent and then more data handled (which will never be sent),
	// the client wrongly succeeds. TODO: figure out the implications of this.
	tlsConfig.MaxVersion = tls.VersionTLS12

	h := fakeirc.NewHelper(ctx, t, fakeirc.TLS(tlsConfig))
	defer h.StopServer()
	serverMessages := h.CollectFromServer()

	d := birc.Dialer{
		Addr:      h.Addr(),
		TLSConfig: fakeirc.TLSConfig,
	}

	_, err := d.Dial(ctx)
	assert.ErrorContains(t, err, "bad certificate")

	assert.ErrorContains(t, h.StopServerErr(), "certificate signed by unknown authority")
	h.AssertMessages(serverMessages)
}
