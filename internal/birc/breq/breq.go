// Package breq implements requests passed around internally inside of birc.
// It has been split out into its own package for testing.
package breq

import (
	"context"
	"sync"

	"github.com/hortbot/hortbot/internal/pkg/ircx"
	"github.com/jakebailey/irc"
)

var errChanPool = sync.Pool{
	New: func() interface{} {
		return make(chan error, 1)
	},
}

func getErrChan() chan error {
	return errChanPool.Get().(chan error)
}

type Send struct {
	errChan chan error
	M       *irc.Message
}

func NewSend(m *irc.Message) Send {
	return Send{
		errChan: getErrChan(),
		M:       m,
	}
}

func (r Send) Do(ctx context.Context, reqChan chan<- Send, stopChan <-chan struct{}, stopErr error) error {
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

func (r Send) Finish(err error) {
	r.errChan <- err
}

type JoinPart struct {
	errChan chan error
	Channel string
	Join    bool
	Force   bool
}

func NewJoinPart(channel string, join, force bool) JoinPart {
	return JoinPart{
		errChan: getErrChan(),
		Channel: ircx.NormalizeChannel(channel),
		Join:    join,
		Force:   force,
	}
}

func (r JoinPart) Do(ctx context.Context, reqChan chan<- JoinPart, stopChan <-chan struct{}, stopErr error) error {
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

func (r JoinPart) Finish(err error) {
	r.errChan <- err
}

type SyncJoined struct {
	errChan  chan error
	Channels []string
}

func NewSyncJoined(channels []string) SyncJoined {
	return SyncJoined{
		errChan:  getErrChan(),
		Channels: ircx.NormalizeChannels(channels...),
	}
}

func (r SyncJoined) Do(ctx context.Context, reqChan chan<- SyncJoined, stopChan <-chan struct{}, stopErr error) error {
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

func (r SyncJoined) Finish(err error) {
	r.errChan <- err
}
