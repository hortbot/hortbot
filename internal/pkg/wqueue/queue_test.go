package wqueue_test

import (
	"context"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/fortytw2/leaktest"
	"github.com/hortbot/hortbot/internal/pkg/assertx"
	"github.com/hortbot/hortbot/internal/pkg/errgroupx"
	"github.com/hortbot/hortbot/internal/pkg/wqueue"
	"gotest.tools/v3/assert"
)

func TestQueueBadLimit(t *testing.T) {
	t.Parallel()

	assertx.Panic(t, func() {
		wqueue.NewQueue[string](0)
	}, "bad limit")

	assertx.Panic(t, func() {
		wqueue.NewQueue[string](-999)
	}, "bad limit")
}

func TestQueueNoWorker(t *testing.T) {
	t.Parallel()

	q := wqueue.NewQueue[string](10)
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	const key = "key"
	var count int
	fn := func(attach wqueue.Attacher) {
		count++
	}

	// This succeeds even without a worker, since the queue won't be full.
	assert.NilError(t, q.Put(ctx, key, fn))
	assert.NilError(t, q.Put(ctx, key, fn))

	time.Sleep(10 * time.Millisecond)

	assert.Equal(t, count, 0)
}

func TestQueueHitLimit(t *testing.T) {
	t.Parallel()

	q := wqueue.NewQueue[string](2)
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	const key = "key"
	var count int
	fn := func(attach wqueue.Attacher) {
		count++
	}

	assert.NilError(t, q.Put(ctx, key, fn))
	assert.NilError(t, q.Put(ctx, key, fn))

	assert.Equal(t, q.Put(ctx, key, fn), context.DeadlineExceeded)
	assert.Equal(t, count, 0)
}

func TestQueueNilWork(t *testing.T) {
	q := wqueue.NewQueue[string](2)
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	assertx.Panic(t, func() {
		_ = q.Put(ctx, "", nil)
	}, "nil WorkFunc")
}

func TestQueue(t *testing.T) {
	defer leaktest.Check(t)()

	q := wqueue.NewQueue[string](10)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	g := errgroupx.FromContext(ctx)
	g.Go(q.Worker)
	g.Go(q.Worker)
	g.Go(q.Worker)
	g.Go(q.Worker)

	ch := make(chan int)
	var count int
	const key = "key"
	fn := func(attach wqueue.Attacher) {
		ctx, cancel := attach(ctx)
		defer cancel()

		select {
		case ch <- count:
			count++
		case <-ctx.Done():
		}
	}

	ch2 := make(chan int)
	var count2 int
	const key2 = "key2"
	fn2 := func(attach wqueue.Attacher) {
		ctx, cancel := attach(context.Background())
		defer cancel()

		select {
		case ch2 <- count2:
			count2++
		case <-ctx.Done():
			// This one can only be canceled by the worker's context.
		}
	}

	// This succeeds even without a worker, since the queue won't be full.
	assert.NilError(t, q.Put(ctx, key, fn))
	assert.NilError(t, q.Put(ctx, key2, fn2))
	assert.Equal(t, <-ch, 0)

	assert.NilError(t, q.Put(ctx, key, fn))
	assert.NilError(t, q.Put(ctx, key, fn))
	assert.NilError(t, q.Put(ctx, key, fn))
	assert.Equal(t, <-ch, 1)
	assert.Equal(t, <-ch, 2)
	assert.Equal(t, <-ch, 3)

	assert.Equal(t, <-ch2, 0)

	// Sleep to allow the queues to move to the empty set.
	time.Sleep(100 * time.Millisecond)
	assert.NilError(t, q.Put(ctx, key, fn))
	assert.NilError(t, q.Put(ctx, key2, fn2))

	// Allow the workers to hang, leakcheck will verify that they close.
	time.Sleep(50 * time.Millisecond)

	g.Stop()
	_ = g.Wait()
}

func TestQueuePanic(t *testing.T) {
	defer leaktest.Check(t)()

	q := wqueue.NewQueue[string](10)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	g := errgroupx.FromContext(ctx)
	g.Go(q.Worker)
	g.Go(q.Worker)
	g.Go(q.Worker)
	g.Go(q.Worker)

	ch := make(chan int)
	var count int
	const key = "key"
	fn := func(attach wqueue.Attacher) {
		ctx, cancel := attach(ctx)
		defer cancel()

		select {
		case ch <- count:
			count++
		case <-ctx.Done():
		}
	}
	panicker := func(attach wqueue.Attacher) {
		panic("uh oh")
	}

	assert.NilError(t, q.Put(ctx, key, fn))
	assert.NilError(t, q.Put(ctx, key, panicker))
	assert.NilError(t, q.Put(ctx, key, fn))
	assert.Equal(t, <-ch, 0)
	assert.Equal(t, <-ch, 1)

	g.Stop()
	_ = g.Wait()
}

func TestQueueStress(t *testing.T) {
	// Stupid attempt at reducing coverage flakes.
	for i := 0; i < 20; i++ {
		testQueueStress(t)
	}
}

func testQueueStress(t *testing.T) { //nolint:thelper
	defer leaktest.Check(t)()

	q := wqueue.NewQueue[string](1000)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	g := errgroupx.FromContext(ctx)
	for i := 0; i < runtime.GOMAXPROCS(0); i++ {
		g.Go(q.Worker)
	}

	const (
		N = 4
		M = 1000
	)
	var counts [N]int
	var wg sync.WaitGroup
	wg.Add(N + N*M) // N putters, N*M subtasks.

	for i := 0; i < N; i++ {
		i := i
		key := string(rune('a' + i))
		g.Go(func(ctx context.Context) error {
			defer wg.Done()

			for j := 0; j < M; j++ {
				if err := q.Put(ctx, key, func(attach wqueue.Attacher) {
					defer wg.Done()
					counts[i]++
				}); err != nil {
					return err
				}
			}

			return nil
		})
	}

	wg.Wait()
	g.Stop()
	_ = g.Wait()
	assert.NilError(t, ctx.Err())

	for i, v := range counts {
		assert.Equal(t, v, M, "count %d", i)
	}
}

func TestQueueStressCancel(t *testing.T) {
	defer leaktest.Check(t)()

	q := wqueue.NewQueue[string](1000)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	g := errgroupx.FromContext(ctx)
	for i := 0; i < runtime.GOMAXPROCS(0); i++ {
		g.Go(q.Worker)
	}

	const (
		N = 4
		M = 1000
	)

	for i := 0; i < N; i++ {
		key := string(rune('a' + i))
		g.Go(func(ctx context.Context) error {
			for j := 0; j < M; j++ {
				j := j
				if err := q.Put(ctx, key, func(attach wqueue.Attacher) {
					if j == M/2 {
						cancel()
					}
				}); err != nil {
					return err
				}
			}

			return nil
		})
	}

	<-ctx.Done()
	g.Stop()
	_ = g.Wait()
	assert.Equal(t, ctx.Err(), context.Canceled)
}
