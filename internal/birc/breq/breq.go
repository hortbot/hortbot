// Package breq implements requests passed around internally inside of birc.
// It has been split out into its own package for testing.
package breq

import (
	"context"

	"github.com/hortbot/hortbot/internal/pkg/correlation"
	"github.com/hortbot/hortbot/internal/pkg/ircx"
	"github.com/hortbot/hortbot/internal/pkg/pool"
	"github.com/jakebailey/irc"
	"github.com/rs/xid"
)

// TODO: In the far future where Go has generics, drop all of this in favor of a single generic implementation.

var errChanPool = pool.NewPool(func() chan error {
	return make(chan error, 1)
})

// Send is a single send request.
type Send struct {
	errChan chan error
	M       *irc.Message
	XID     xid.ID
}

// NewSend creates a send request for the given message.
func NewSend(m *irc.Message) Send {
	return Send{
		errChan: errChanPool.Get(),
		M:       m,
	}
}

// Do sends the request over the request channel and waits for the request to be completed.
// If the request is canceled, then the context error will be returned and the request ignored.
// If stopChan closes, then stopErr will be returned and the request ignored.
func (r Send) Do(ctx context.Context, reqChan chan<- Send, stopChan <-chan struct{}, stopErr error) error {
	r.XID = correlation.FromContext(ctx)

	select {
	case reqChan <- r:
		// Do nothing.
	case <-ctx.Done():
		return ctx.Err()
	case <-stopChan:
		return stopErr
	}

	select {
	case err := <-r.errChan:
		errChanPool.Put(r.errChan)
		return err
	case <-ctx.Done():
		return ctx.Err()
	case <-stopChan:
		return stopErr
	}
}

// Finish completes the request with the given error, which may be nil.
func (r Send) Finish(err error) {
	r.errChan <- err
}

// JoinPart is a request for a JOIN or PART.
type JoinPart struct {
	errChan chan error
	Channel string
	Join    bool
	XID     xid.ID
}

// NewJoinPart creates a new JOIN/PART request.
func NewJoinPart(channel string, join bool) JoinPart {
	return JoinPart{
		errChan: errChanPool.Get(),
		Channel: ircx.NormalizeChannel(channel),
		Join:    join,
	}
}

// Do sends the request over the request channel and waits for the request to be completed.
// If the request is canceled, then the context error will be returned and the request ignored.
// If stopChan closes, then stopErr will be returned and the request ignored.
func (r JoinPart) Do(ctx context.Context, reqChan chan<- JoinPart, stopChan <-chan struct{}, stopErr error) error {
	r.XID = correlation.FromContext(ctx)

	select {
	case reqChan <- r:
		// Do nothing.
	case <-ctx.Done():
		return ctx.Err()
	case <-stopChan:
		return stopErr
	}

	select {
	case err := <-r.errChan:
		errChanPool.Put(r.errChan)
		return err
	case <-ctx.Done():
		return ctx.Err()
	case <-stopChan:
		return stopErr
	}
}

// Finish completes the request with the given error, which may be nil.
func (r JoinPart) Finish(err error) {
	r.errChan <- err
}

// SyncJoined is a request to sync the joined channels with a given list.
type SyncJoined struct {
	errChan  chan error
	Channels []string
	XID      xid.ID
}

// NewSyncJoined creates a new sync-join request.
func NewSyncJoined(channels []string) SyncJoined {
	return SyncJoined{
		errChan:  errChanPool.Get(),
		Channels: ircx.NormalizeChannels(channels...),
	}
}

// Do sends the request over the request channel and waits for the request to be completed.
// If the request is canceled, then the context error will be returned and the request ignored.
// If stopChan closes, then stopErr will be returned and the request ignored.
func (r SyncJoined) Do(ctx context.Context, reqChan chan<- SyncJoined, stopChan <-chan struct{}, stopErr error) error {
	r.XID = correlation.FromContext(ctx)

	select {
	case reqChan <- r:
		// Do nothing.
	case <-ctx.Done():
		return ctx.Err()
	case <-stopChan:
		return stopErr
	}

	select {
	case err := <-r.errChan:
		errChanPool.Put(r.errChan)
		return err
	case <-ctx.Done():
		return ctx.Err()
	case <-stopChan:
		return stopErr
	}
}

// Finish completes the request with the given error, which may be nil.
func (r SyncJoined) Finish(err error) {
	r.errChan <- err
}
