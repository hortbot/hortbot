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

var errChanPool = pool.NewPool(func() chan error {
	return make(chan error, 1)
})

type Request[T any] struct {
	errChan chan error
	Data    T
	XID     xid.ID
}

func newRequest[T any](data T) Request[T] {
	return Request[T]{
		errChan: errChanPool.Get(),
		Data:    data,
	}
}

// Do sends the request over the request channel and waits for the request to be completed.
// If the request is canceled, then the context error will be returned and the request ignored.
// If stopChan closes, then stopErr will be returned and the request ignored.
func (r Request[T]) Do(ctx context.Context, reqChan chan<- Request[T], stopChan <-chan struct{}, stopErr error) error {
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
func (r Request[T]) Finish(err error) {
	r.errChan <- err
}

// Send is a single send request.
type Send = Request[*irc.Message]

func NewSend(m *irc.Message) Send {
	return newRequest(m)
}

// JoinPart is a request for a JOIN or PART.
type JoinPart = Request[struct {
	Channel string
	Join    bool
}]

// NewJoinPart creates a new JOIN/PART request.
func NewJoinPart(channel string, join bool) JoinPart {
	return newRequest(struct {
		Channel string
		Join    bool
	}{
		Channel: ircx.NormalizeChannel(channel),
		Join:    join,
	})
}

// SyncJoined is a request to sync the joined channels with a given list.
type SyncJoined = Request[[]string]

// NewSyncJoined creates a new sync-join request.
func NewSyncJoined(channels []string) SyncJoined {
	return newRequest(ircx.NormalizeChannels(channels...))
}
