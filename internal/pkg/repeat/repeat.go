package repeat

import (
	"context"
	"sync"
	"time"

	"github.com/hortbot/hortbot/internal/pkg/errgroupx"
	"github.com/leononame/clock"
	"github.com/robfig/cron/v3"
)

type Repeater struct {
	clock clock.Clock

	reps  *taskRunner
	crons *taskRunner
}

// New creates a new Repeater with the specified root context and clock.
// If ctx is nil, then context.Background() will be used instead.
func New(ctx context.Context, clock clock.Clock) *Repeater {
	if ctx == nil {
		ctx = context.Background()
	}

	return &Repeater{
		clock: clock,
		reps:  newTaskRunner(ctx),
		crons: newTaskRunner(ctx),
	}
}

// Stop stops the repeater, cancelling all running tasks and waiting for all
// to return.
func (r *Repeater) Stop() {
	r.reps.stop()
	r.crons.stop()
	r.reps.wait()
	r.crons.wait()
}

// Add adds a repeated task occurring at an interval, given the specified ID.
// If init is non-zero, then the task will first wait for that duration before
// looping. If there is already a task with that ID, then it will be replaced.
func (r *Repeater) Add(id int64, fn func(ctx context.Context, id int64), interval, init time.Duration) {
	r.reps.run(id, func(ctx context.Context) {
		if init != 0 {
			select {
			case <-ctx.Done():
				return
			case <-r.clock.After(init):
			}
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
	r.reps.remove(id)
}

// AddCron adds a repeated task which repeats based on a cron expression.
// Cron tasks may safely share IDs with regular repeated tasks.
func (r *Repeater) AddCron(id int64, fn func(ctx context.Context, id int64), expr cron.Schedule) {
	r.crons.run(id, func(ctx context.Context) {
		for {
			now := r.clock.Now()
			next := expr.Next(now)
			if next.IsZero() {
				return
			}

			dur := next.Sub(now)
			// fmt.Println(now, next, dur)
			// if dur < time.Second {
			// 	panic("wtf")
			// }

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
	t.g.Wait() //nolint:errcheck
}

var cronParser = cron.NewParser(cron.SecondOptional | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor)

func ParseCron(s string) (cron.Schedule, error) {
	return cronParser.Parse(s)
}
