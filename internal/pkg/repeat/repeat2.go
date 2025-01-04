package repeat

import (
	"container/heap"
	"context"
	"time"

	"github.com/hortbot/hortbot/internal/pkg/errgroupx"
	"github.com/leononame/clock"
	"github.com/robfig/cron/v3"
)

type ident struct {
	id   int64
	cron bool
}

type job struct {
	ident ident
	run   func(context.Context, int64) (readd bool)
	next  func(time.Time) time.Time
}

func repeatJob(id int64, run func(context.Context, int64) bool, start time.Time, interval time.Duration) *job {
	return &job{
		ident: ident{id: id},
		run:   run,
		next: func(now time.Time) time.Time {
			n := now.Sub(start)/interval + 1 // Ceiling[(now - start) / interval]
			return start.Add(n * interval)
		},
	}
}

func cronJob(id int64, run func(context.Context, int64) bool, sched cron.Schedule) *job {
	return &job{
		ident: ident{id: id, cron: true},
		run:   run,
		next:  sched.Next,
	}
}

type count struct {
	reps  int
	crons int
}

func (c count) Add(other count) count {
	return count{
		reps:  c.reps + other.reps,
		crons: c.crons + other.crons,
	}
}

type manager struct {
	clock       clock.Clock
	queue       queue
	identToItem map[ident]*item

	addChan    chan *job
	removeChan chan ident
	resetChan  chan struct{}
	countChan  chan count

	counts count
}

func newManager(clock clock.Clock) *manager {
	return &manager{
		clock:       clock,
		identToItem: make(map[ident]*item),
		addChan:     make(chan *job),
		removeChan:  make(chan ident),
		resetChan:   make(chan struct{}),
		countChan:   make(chan count),
	}
}

func (m *manager) run(ctx context.Context) error {
	var (
		closedTimeChan = make(chan time.Time)
		currReady      <-chan time.Time
		currItem       *item
		currCount      count
	)

	// Ensure reads from this channel always return immediately.
	close(closedTimeChan)

	g := errgroupx.FromContext(ctx)

	for {
		select {
		case <-ctx.Done():
			_ = g.Wait()
			return ctx.Err()

		case <-currReady:
			job := currItem.job
			g.Go(func(ctx context.Context) error {
				if job.run(ctx, job.ident.id) {
					return m.add(ctx, job)
				}
				return ctx.Err()
			})

			// Exit select and find a new job.

		case job := <-m.addChan:
			id := job.ident
			now := m.clock.Now()
			newDeadline := job.next(now)

			if newDeadline.IsZero() {
				continue
			}

			if i := m.identToItem[id]; i != nil {
				i.job = job
				i.deadline = newDeadline
				m.queueFix(i)
			} else {
				m.queuePush(&item{
					deadline: newDeadline,
					job:      job,
				})
			}

			if currItem == nil {
				// No current item, break to select one.
				// TODO: Is this always the above item? Worth optimizing?
				break
			}

			if currItem.job.ident == id {
				// Updated the current item, so it's no longer valid. Select a new one.
				break
			}

			if currItem.deadline.Before(newDeadline) {
				// Current item is still the closest, just continue to wait for it.
				continue
			}

			// Added item is better, just push the old item back and let the code below fetch the new one.
			m.queuePush(currItem)

		case id := <-m.removeChan:
			if currItem != nil && currItem.job.ident == id {
				// Current item is the one being removed, break and pop the next one.
				break
			}
			m.queueRemove(id)
			continue

		case <-m.resetChan:
			m.queueReset()

		case m.countChan <- m.counts.Add(currCount):
			continue
		}

		// Find a new job.
		// currReady and currItem must be set before continuing.

		var ok bool
		currItem, ok = m.queuePop()
		if !ok {
			currItem = nil
			currReady = nil
			currCount = count{}
			continue
		}

		if currItem.job.ident.cron {
			currCount = count{crons: 1}
		} else {
			currCount = count{reps: 1}
		}

		delay := m.clock.Until(currItem.deadline)
		if delay > 0 {
			currReady = m.clock.After(delay)
		} else {
			// Already past the deadline, force the next iteration to run the job.
			currReady = closedTimeChan
		}
	}
}

func (m *manager) add(ctx context.Context, j *job) error {
	select {
	case m.addChan <- j:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (m *manager) remove(ctx context.Context, ident ident) error {
	select {
	case m.removeChan <- ident:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (m *manager) count(ctx context.Context) (repeats, schedules int, err error) {
	select {
	case c := <-m.countChan:
		return c.reps, c.crons, nil
	case <-ctx.Done():
		return 0, 0, ctx.Err()
	}
}

func (m *manager) reset(ctx context.Context) error {
	select {
	case m.resetChan <- struct{}{}:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (m *manager) queuePush(i *item) {
	id := i.job.ident

	heap.Push(&m.queue, i)
	m.identToItem[id] = i

	if id.cron {
		m.counts.crons++
	} else {
		m.counts.reps++
	}
}

func (m *manager) queuePop() (*item, bool) {
	if m.queue.Len() == 0 {
		return nil, false
	}

	i := heap.Pop(&m.queue).(*item)

	delete(m.identToItem, i.job.ident)
	if i.job.ident.cron {
		m.counts.crons--
	} else {
		m.counts.reps--
	}

	return i, true
}

func (m *manager) queueRemove(id ident) {
	i := m.identToItem[id]
	if i == nil {
		return
	}

	heap.Remove(&m.queue, i.index)
	delete(m.identToItem, id)

	if id.cron {
		m.counts.crons--
	} else {
		m.counts.reps--
	}
}

func (m *manager) queueFix(i *item) {
	heap.Fix(&m.queue, i.index)
}

func (m *manager) queueReset() {
	if len(m.queue) != 0 {
		m.queue = m.queue[:0]
		m.identToItem = make(map[ident]*item)
		m.counts = count{}
	}
}

// A priority queue sorted by deadline.
type queue []*item //nolint:recvcheck

type item struct {
	index    int
	deadline time.Time
	job      *job
}

func (q queue) Len() int {
	return len(q)
}

func (q queue) Less(i, j int) bool {
	d1 := q[i].deadline
	d2 := q[j].deadline
	return d1.Before(d2)
}

func (q queue) Swap(i, j int) {
	q[i], q[j] = q[j], q[i]
	q[i].index = i
	q[j].index = j
}

func (q *queue) Push(x any) {
	item := x.(*item)
	item.index = len(*q)
	*q = append(*q, item)
}

func (q *queue) Pop() any {
	old := *q
	n := len(old)
	item := old[n-1]
	old[n-1] = nil
	item.index = -1
	*q = old[0 : n-1]
	return item
}

var _ heap.Interface = (*queue)(nil)
