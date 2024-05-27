// Package repeat provides a system to run functions repeated on a schedule.
package repeat

import (
	"context"
	"time"

	"github.com/leononame/clock"
	"github.com/robfig/cron/v3"
)

// Repeater manages repeated functions.
type Repeater struct {
	m *manager
}

// New creates a new Repeater with the specified clock.
func New(clock clock.Clock) *Repeater {
	return &Repeater{
		m: newManager(clock),
	}
}

// Run runs the repeater until the context is canceled.
func (r *Repeater) Run(ctx context.Context) error {
	return r.m.run(ctx)
}

// Reset resets a repeater, cancelling and removing all of its repeated functions.
func (r *Repeater) Reset(ctx context.Context) error {
	return r.m.reset(ctx)
}

// Count returns the number of repeated and scheduled functions.
func (r *Repeater) Count(ctx context.Context) (repeats, schedules int, err error) {
	return r.m.count(ctx)
}

// Add adds a repeated task occurring at an interval, given the specified ID.
func (r *Repeater) Add(ctx context.Context, id int64, fn func(ctx context.Context, id int64) (readd bool), start time.Time, interval time.Duration) error {
	return r.m.add(ctx, repeatJob(id, fn, start, interval))
}

// Remove removes a repeated task.
func (r *Repeater) Remove(ctx context.Context, id int64) error {
	return r.m.remove(ctx, ident{id: id})
}

// AddCron adds a repeated task which repeats based on a cron expression.
// Cron tasks may safely share IDs with regular repeated tasks.
func (r *Repeater) AddCron(ctx context.Context, id int64, fn func(ctx context.Context, id int64) (readd bool), expr *Cron) error {
	return r.m.add(ctx, cronJob(id, fn, expr.expr))
}

// RemoveCron removes a repeated cron task.
func (r *Repeater) RemoveCron(ctx context.Context, id int64) error {
	return r.m.remove(ctx, ident{id: id, cron: true})
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
		return nil, err //nolint:wrapcheck
	}

	return &Cron{
		expr: expr,
	}, nil
}
