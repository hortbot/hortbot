// Package wqueue implements a work queue, with independent internal subqueues.
//
// Inspired by Brian C. Mills' "Rethinking Classical Concurrency Patterns"
// talk, specifically the queue that passes data between putters and workers
// to coordinate state changes between channels.
package wqueue

import (
	"context"

	"github.com/zikaeroh/ctxlog"
	"go.uber.org/zap"
)

// Queue is a work queue, consisting of multiple subqueues which can operate
// independently while sharing the same workers.
type Queue[K comparable] struct {
	// States the state machine can be in.
	noWork         chan *state[K] // Empty, or all subqueues are locked.
	noWorkLimited  chan *state[K] // All subqueues are locked, and the number of items is at the limit.
	hasWork        chan *state[K] // There may be work to do.
	hasWorkLimited chan *state[K] // There may be work to do, and the number of items is at the limit.
}

// NewQueue creates a new Queue which can grow to a maximum size of limit.
func NewQueue[K comparable](limit int) *Queue[K] {
	if limit <= 0 {
		panic("bad limit")
	}

	q := &Queue[K]{
		noWork:         make(chan *state[K], 1),
		noWorkLimited:  make(chan *state[K], 1),
		hasWork:        make(chan *state[K], 1),
		hasWorkLimited: make(chan *state[K], 1),
	}

	// Seed with some initial state.
	q.noWork <- &state[K]{
		empty:     make(map[K]*subQueue),
		unlocked:  newOrderedSet[K, *subQueue](),
		locked:    make(map[K]*subQueue),
		sizeLimit: limit,
	}

	return q
}

// WorkFunc is a function called by a worker. The attach function can be used
// to attach the canceleation of the worker to another context.
type WorkFunc func(attach Attacher)

// Put puts the worker function into the keyed subqueue. This function will
// be called at some point in the future by a worker, but never concurrently
// with other items in the keyed subqueue.
func (q *Queue[K]) Put(ctx context.Context, key K, fn WorkFunc) error {
	if fn == nil {
		panic("nil WorkFunc")
	}

	// Get the current state so long as we aren't limited.
	state, err := q.getForPut(ctx)
	if err != nil {
		return err
	}

	state.addWork(key, fn)

	return q.putWork(ctx, state)
}

// Worker runs items from the queues, exiting when the context has been canceled.
func (q *Queue[K]) Worker(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	attach := attachFunc(ctx)

	for {
		var work WorkFunc
		var key K

		// Use a block here to enforce that the below state does not get reused once put.
		{
			// Get the current state so long as there is work to do.
			state, err := q.getForWork(ctx)
			if err != nil {
				return err
			}

			work, key = state.findWork()

			if work == nil {
				// No actionable work found, either because there is none or it's all locked.
				if err := q.putNoWork(ctx, state); err != nil {
					return err
				}
				continue
			}

			if err := q.putWork(ctx, state); err != nil {
				return err
			}
		}

		func() {
			defer func() {
				if r := recover(); r != nil {
					ctxlog.Error(ctx, "work function panicked", zap.Any("key", key), zap.Any("value", r))
				}
			}()
			work(attach)
		}()

		// The context may have been canceled while the work was running, but the next
		// call will check that and return.

		{
			// Get the state unconditionally, in order to get permission to modify the state.
			state, err := q.getAny(ctx)
			if err != nil {
				return err
			}

			state.unlock(key)

			if err := q.putWork(ctx, state); err != nil {
				return err
			}
		}
	}
}

func (q *Queue[K]) getForPut(ctx context.Context) (*state[K], error) {
	return getState2(ctx, q.noWork, q.hasWork)
}

func (q *Queue[K]) getForWork(ctx context.Context) (*state[K], error) {
	return getState2(ctx, q.hasWork, q.hasWorkLimited)
}

func (q *Queue[K]) getAny(ctx context.Context) (*state[K], error) {
	return getState4(ctx, q.noWork, q.noWorkLimited, q.hasWork, q.hasWorkLimited)
}

func (q *Queue[K]) putNoWork(ctx context.Context, state *state[K]) error {
	return putState(ctx, state, q.noWork, q.noWorkLimited)
}

func (q *Queue[K]) putWork(ctx context.Context, state *state[K]) error {
	return putState(ctx, state, q.hasWork, q.hasWorkLimited)
}

type subQueue struct {
	items []WorkFunc
}

// The contents of this could be on Queue itself (with the states channels
// being the empty struct), but having a dedicated type for this helps enforce
// the invariant that only one goroutine can get, modify, and put this state at
// once.
type state[K comparable] struct {
	empty           map[K]*subQueue           // len(items) == 0
	unlocked        *orderedSet[K, *subQueue] // len(items) != 0 and no worker is operating on this subqueue
	locked          map[K]*subQueue           // len(items) != 0 and a worker is operating on this subqueue
	size, sizeLimit int
}

func (s *state[K]) addWork(key K, work WorkFunc) {
	subQ := s.empty[key]
	if subQ == nil {
		subQ = s.locked[key]
		if subQ == nil {
			subQ = s.unlocked.find(key)
			if subQ == nil {
				subQ = &subQueue{}
				s.unlocked.add(key, subQ)
			}
		}
	} else {
		// We're about to add work to this subqueue, so it can't be in the empty set.
		// An empty subqueue cannot be locked, so move it to the unlocked set.
		delete(s.empty, key)
		s.unlocked.add(key, subQ)
	}

	subQ.items = append(subQ.items, work)
	s.size++
}

func (s *state[K]) findWork() (fn WorkFunc, key K) {
	key, subQ := s.unlocked.next()

	if subQ == nil {
		var zero K
		return nil, zero
	}

	s.size--
	fn = subQ.items[0]
	subQ.items[0] = nil // Prevent leaks.
	subQ.items = subQ.items[1:]
	s.locked[key] = subQ
	return fn, key
}

func (s *state[K]) unlock(key K) {
	subQ := s.locked[key] // Should never be nil.
	delete(s.locked, key)

	if len(subQ.items) == 0 {
		s.empty[key] = subQ
	} else {
		s.unlocked.add(key, subQ) // Put newly unlocked queue at the end.
	}
}

func (s *state[_]) isLimited() bool {
	return s.size >= s.sizeLimit
}

func getState2[K comparable](ctx context.Context, a, b chan *state[K]) (state *state[K], err error) {
	select {
	case state = <-a:
	case state = <-b:
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	return state, nil
}

func getState4[K comparable](ctx context.Context, a, b, c, d chan *state[K]) (state *state[K], err error) {
	select {
	case state = <-a:
	case state = <-b:
	case state = <-c:
	case state = <-d:
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	return state, nil
}

func putState[K comparable](ctx context.Context, state *state[K], ch chan *state[K], limited chan *state[K]) error {
	if state.isLimited() {
		ch = limited
	}

	select {
	case ch <- state:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
