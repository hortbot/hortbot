package bnsq

import (
	"sync"
	"testing"

	"github.com/nsqio/nsq/nsqd"
)

type nilLogger struct{}

func (l nilLogger) Output(_ int, _ string) error {
	return nil
}

// NewTestNSQD starts an in-process nsqd for testing and returns its TCP address.
// The server is automatically stopped when the test completes.
func NewTestNSQD(t testing.TB) string {
	t.Helper()
	addr, _ := newTestNSQD(t)
	return addr
}

// NewTestNSQDWithStop starts an in-process nsqd and returns its TCP address
// and a function to stop it early. The server is also stopped on test cleanup.
func NewTestNSQDWithStop(t testing.TB) (string, func()) {
	t.Helper()
	return newTestNSQD(t)
}

func newTestNSQD(t testing.TB) (string, func()) {
	t.Helper()

	opts := nsqd.NewOptions()
	opts.TCPAddress = "127.0.0.1:0"
	opts.HTTPAddress = "127.0.0.1:0"
	opts.Logger = nilLogger{}

	opts.DataPath = t.TempDir()

	n, err := nsqd.New(opts)
	if err != nil {
		t.Fatalf("creating nsqd: %v", err)
	}

	go func() {
		_ = n.Main()
	}()

	stop := sync.OnceFunc(func() {
		n.Exit()
	})

	t.Cleanup(stop)

	return n.RealTCPAddr().String(), stop
}
