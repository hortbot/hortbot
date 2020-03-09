package wqueue

import (
	"context"
	"sync"
	"time"
)

// An Attacher attaches an extra cancelletion to the provided context.
type Attacher func(ctx context.Context) (context.Context, context.CancelFunc)

type joinedContext struct {
	base  context.Context
	extra context.Context

	doneOnce sync.Once
	done     chan struct{}

	errMu sync.Mutex
	err   error
}

func (j *joinedContext) Deadline() (deadline time.Time, ok bool) {
	deadline, ok = j.base.Deadline()
	if !ok {
		return j.extra.Deadline()
	}

	extra, ok := j.extra.Deadline()
	if ok && extra.Before(deadline) {
		return extra, true
	}

	return deadline, true
}

func (j *joinedContext) Done() <-chan struct{} {
	// Lazily start the goroutine when this channel is requested.
	// We can create the done chan from the parent contexts at any time.
	j.doneOnce.Do(j.runCloser)
	return j.done
}

func (j *joinedContext) runCloser() {
	j.done = make(chan struct{})
	go func() {
		defer close(j.done)

		var err error
		select {
		case <-j.base.Done():
			err = j.base.Err()
		case <-j.extra.Done():
			err = j.extra.Err()
		}

		j.errMu.Lock()
		defer j.errMu.Unlock()
		j.err = err
	}()
}

func (j *joinedContext) Err() error {
	j.errMu.Lock()
	defer j.errMu.Unlock()
	return j.err
}

func (j *joinedContext) Value(key interface{}) interface{} {
	return j.base.Value(key)
}

func attachFunc(ctx context.Context) Attacher {
	if ctx.Done() == nil {
		return func(ctx context.Context) (context.Context, context.CancelFunc) {
			return ctx, func() {}
		}
	}

	return func(base context.Context) (context.Context, context.CancelFunc) {
		base, cancel := context.WithCancel(base)
		return &joinedContext{
			base:  base,
			extra: ctx,
		}, cancel
	}
}
