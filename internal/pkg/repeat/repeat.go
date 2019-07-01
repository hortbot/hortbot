package repeat

import (
	"context"
	"sync"
	"time"

	"github.com/efritz/glock"
	"github.com/hortbot/hortbot/internal/pkg/errgroupx"
)

type Repeater struct {
	g     *errgroupx.Group
	clock glock.Clock

	mu          sync.Mutex
	cancelFuncs map[int64]func()
}

// New creates a new Repeater with the specified root context and clock.
// If ctx is nil, then context.Background() will be used instead.
func New(ctx context.Context, clock glock.Clock) *Repeater {
	if ctx == nil {
		ctx = context.Background()
	}

	return &Repeater{
		g:           errgroupx.FromContext(ctx),
		clock:       clock,
		cancelFuncs: make(map[int64]func()),
	}
}

// Stop stops the repeater, cancelling all running tasks and waiting for all
// to return.
func (r *Repeater) Stop() {
	r.g.Stop()
	r.g.Wait() //nolint:errcheck
}

// Add adds a repeated task occurring at an interval, given the specified ID.
// If init is non-zero, then the task will first wait for that duration before
// looping. If there is already a task with that ID, then it will be replaced.
func (r *Repeater) Add(id int64, fn func(ctx context.Context, id int64), interval, init time.Duration) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.remove(id)

	ready := make(chan struct{})

	r.g.Go(func(ctx context.Context) error {
		ctx, cancel := context.WithCancel(ctx)
		defer cancel()

		r.cancelFuncs[id] = cancel
		close(ready)

		if init != 0 {
			select {
			case <-ctx.Done():
				return nil
			case <-r.clock.After(init):
			}
		}

		ticker := r.clock.NewTicker(interval)
		defer ticker.Stop()
		tick := ticker.Chan()

		for {
			select {
			case <-ctx.Done():
				return nil

			case <-tick:
				fn(ctx, id)
			}
		}
	})

	<-ready
}

// Remove removes a task.
func (r *Repeater) Remove(id int64) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.remove(id)
}

func (r *Repeater) remove(id int64) {
	if cancel := r.cancelFuncs[id]; cancel != nil {
		delete(r.cancelFuncs, id)
		cancel()
	}
}
