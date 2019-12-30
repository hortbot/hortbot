package dnsq

import "testing"

func TestNopDelegate(t *testing.T) {
	var delegate *nopDelegate

	// Ensure these don't actually do anything.
	delegate.OnResponse(nil, nil)
	delegate.OnError(nil, nil)
	delegate.OnMessage(nil, nil)
	delegate.OnMessageFinished(nil, nil)
	delegate.OnMessageRequeued(nil, nil)
	delegate.OnBackoff(nil)
	delegate.OnContinue(nil)
	delegate.OnResume(nil)
	delegate.OnIOError(nil, nil)
	delegate.OnHeartbeat(nil)
	delegate.OnClose(nil)
}
