// Package repeat provides a system to run functions repeated on a schedule.
package repeat

import (
	"context"
	"sync"
	"time"

	"github.com/hortbot/hortbot/internal/pkg/errgroupx"
	"github.com/leononame/clock"
	"github.com/robfig/cron/v3"
)

// Repeater manages repeated functions.
type Repeater struct {
	clock clock.Clock
	ctx   context.Context

	mu    sync.RWMutex
	reps  *taskRunner
	crons *taskRunner
}

// New creates a new Repeater with the specified root context and clock.
func New(ctx context.Context, clock clock.Clock) *Repeater {
	return &Repeater{
		clock: clock,
		ctx:   ctx,
		reps:  newTaskRunner(ctx),
		crons: newTaskRunner(ctx),
	}
}

// Stop stops the repeater, cancelling all running tasks and waiting for all
// to return.
func (r *Repeater) Stop() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.reps.stop()
	r.crons.stop()
	r.reps.wait()
	r.crons.wait()
}

// Reset resets a repeater, cancelling and removing all of its repeated functions.
func (r *Repeater) Reset() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.reps.stop()
	r.crons.stop()
	r.reps = newTaskRunner(r.ctx)
	r.crons = newTaskRunner(r.ctx)
}

// Count returns the number of repeated and scheduled functions.
func (r *Repeater) Count() (repeats, schedules int) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.reps.count(), r.crons.count()
}

// Add adds a repeated task occurring at an interval, given the specified ID.
// If init is non-zero, then the task will first wait for that duration before
// looping. If there is already a task with that ID, then it will be replaced.
func (r *Repeater) Add(id int64, fn func(ctx context.Context, id int64), interval, init time.Duration) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	r.reps.run(id, func(ctx context.Context) {
		if init != 0 {
			select {
			case <-ctx.Done():
				return
			case <-r.clock.After(init):
			}
			// The ticker below won't immediately tick, so run this now.
			fn(ctx, id)
		}

		ticker := r.clock.NewTicker(interval)
		defer ticker.Stop()
		tick := ticker.Chan()

		for {
			select {
			case <-ctx.Done():
				return

			case <-tick:
				fn(ctx, id)
			}
		}
	})
}

// Remove removes a repeated task.
func (r *Repeater) Remove(id int64) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	r.reps.remove(id)
}

// AddCron adds a repeated task which repeats based on a cron expression.
// Cron tasks may safely share IDs with regular repeated tasks.
func (r *Repeater) AddCron(id int64, fn func(ctx context.Context, id int64), expr *Cron) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	r.crons.run(id, func(ctx context.Context) {
		for {
			now := r.clock.Now()
			next := expr.expr.Next(now)
			if next.IsZero() {
				return
			}

			dur := next.Sub(now)

			select {
			case <-ctx.Done():
				return

			case <-r.clock.After(dur):
				fn(ctx, id)
			}
		}
	})
}

// RemoveCron removes a repeated cron task.
func (r *Repeater) RemoveCron(id int64) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	r.crons.remove(id)
}

type taskRunner struct {
	g *errgroupx.Group

	mu      sync.Mutex
	cancels map[int64]func()
}

func newTaskRunner(ctx context.Context) *taskRunner {
	return &taskRunner{
		g:       errgroupx.FromContext(ctx),
		cancels: make(map[int64]func()),
	}
}

func (t *taskRunner) run(id int64, fn func(ctx context.Context)) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if cancel := t.cancels[id]; cancel != nil {
		cancel()
	}

	ready := make(chan struct{})

	t.g.Go(func(ctx context.Context) error {
		ctx, cancel := context.WithCancel(ctx)
		defer cancel()

		t.cancels[id] = cancel
		close(ready)

		fn(ctx)
		return nil
	})

	<-ready
}

func (t *taskRunner) remove(id int64) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if cancel := t.cancels[id]; cancel != nil {
		delete(t.cancels, id)
		cancel()
	}
}

func (t *taskRunner) stop() {
	t.g.Stop()
}

func (t *taskRunner) wait() {
	_ = t.g.Wait()
}

func (t *taskRunner) count() int {
	t.mu.Lock()
	defer t.mu.Unlock()
	return len(t.cancels)
}

var cronParser = cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor)

// Cron is a cron schedule.
type Cron struct {
	expr cron.Schedule
}

// ParseCron parses a crontab line into a Cron.
func ParseCron(s string) (*Cron, error) {
	expr, err := cronParser.Parse(s)
	if err != nil {
		return nil, err
	}

	return &Cron{
		expr: expr,
	}, nil
}
