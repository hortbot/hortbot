//nolint:paralleltest
package repeat_test

import (
	"context"
	"testing"
	"testing/synctest"
	"time"

	"github.com/hortbot/hortbot/internal/pkg/repeat"
	"gotest.tools/v3/assert"
)

func runRepeat(ctx context.Context, r *repeat.Repeater) <-chan error {
	errCh := make(chan error, 1)
	go func() {
		errCh <- r.Run(ctx)
	}()
	return errCh
}

// advance sleeps for d and then waits for all goroutines in the
// synctest bubble to settle.
func advance(d time.Duration) {
	time.Sleep(d)
	synctest.Wait()
}

func TestDoNothing(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ctx, cancel := context.WithCancel(t.Context())
		defer cancel()

		r := repeat.New()
		errCh := runRepeat(ctx, r)

		cancel()
		synctest.Wait()

		assert.Equal(t, <-errCh, context.Canceled)
	})
}

func TestAdd(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ctx, cancel := context.WithCancel(t.Context())
		defer cancel()

		r := repeat.New()
		errCh := runRepeat(ctx, r)
		synctest.Wait()

		count := 0
		fn := func(ctx context.Context, id int64) bool {
			count++
			return true
		}

		assert.NilError(t, r.Add(ctx, 0, fn, time.Now(), time.Second))
		synctest.Wait()

		advance(100 * time.Millisecond)
		for range 5 {
			advance(time.Second)
		}
		advance(100 * time.Millisecond)

		cancel()
		synctest.Wait()

		assert.Equal(t, <-errCh, context.Canceled)
		assert.Equal(t, count, 5)
	})
}

func TestAddWithInit(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ctx, cancel := context.WithCancel(t.Context())
		defer cancel()

		r := repeat.New()
		errCh := runRepeat(ctx, r)
		synctest.Wait()

		count := 0
		fn := func(ctx context.Context, id int64) bool {
			count++
			return true
		}

		assert.NilError(t, r.Add(ctx, 0, fn, time.Now().Add(-time.Second/2), time.Second))
		synctest.Wait()

		advance(100 * time.Millisecond)
		for range 5 {
			advance(time.Second)
		}
		advance(100 * time.Millisecond)

		cancel()
		synctest.Wait()

		assert.Equal(t, <-errCh, context.Canceled)
		assert.Equal(t, count, 5)
	})
}

func TestAddWithInitCancel(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ctx, cancel := context.WithCancel(t.Context())
		defer cancel()

		r := repeat.New()
		errCh := runRepeat(ctx, r)
		synctest.Wait()

		count := 0
		fn := func(ctx context.Context, id int64) bool {
			count++
			return true
		}

		assert.NilError(t, r.Add(ctx, 0, fn, time.Now().Add(-time.Second/2), time.Second))
		synctest.Wait()

		cancel()
		synctest.Wait()

		// After cancel, advancing time should not trigger callbacks.
		advance(5 * time.Second)

		assert.Equal(t, <-errCh, context.Canceled)
		assert.Equal(t, count, 0)
	})
}

func TestAddTwice(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ctx, cancel := context.WithCancel(t.Context())
		defer cancel()

		r := repeat.New()
		errCh := runRepeat(ctx, r)
		synctest.Wait()

		count := 0
		fn := func(ctx context.Context, id int64) bool {
			count++
			return true
		}

		assert.NilError(t, r.Add(ctx, 0, fn, time.Now(), time.Second))
		synctest.Wait()

		advance(100 * time.Millisecond)
		advance(time.Second)
		advance(time.Second)

		// Re-add with a longer interval; should not fire again for a while.
		assert.NilError(t, r.Add(ctx, 0, fn, time.Now(), time.Minute))
		synctest.Wait()

		for range 3 {
			advance(time.Second)
		}
		advance(100 * time.Millisecond)

		cancel()
		synctest.Wait()

		assert.Equal(t, <-errCh, context.Canceled)
		assert.Equal(t, count, 2)
	})
}

func TestAddRemove(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ctx, cancel := context.WithCancel(t.Context())
		defer cancel()

		r := repeat.New()
		errCh := runRepeat(ctx, r)
		synctest.Wait()

		count := 0
		fn := func(ctx context.Context, id int64) bool {
			count++
			return true
		}

		assert.NilError(t, r.Add(ctx, 0, fn, time.Now(), time.Second))
		synctest.Wait()

		advance(100 * time.Millisecond)
		advance(time.Second)
		advance(time.Second)

		assert.NilError(t, r.Remove(ctx, 0))
		synctest.Wait()

		for range 3 {
			advance(time.Second)
		}
		advance(100 * time.Millisecond)

		cancel()
		synctest.Wait()

		assert.Equal(t, <-errCh, context.Canceled)
		assert.Equal(t, count, 2)
	})
}

func TestAddStop(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ctx, cancel := context.WithCancel(t.Context())
		defer cancel()

		r := repeat.New()
		errCh := runRepeat(ctx, r)
		synctest.Wait()

		count := 0
		fn := func(ctx context.Context, id int64) bool {
			count++
			return true
		}

		assert.NilError(t, r.Add(ctx, 0, fn, time.Now(), time.Second))
		synctest.Wait()

		advance(100 * time.Millisecond)
		advance(time.Second)
		advance(time.Second)

		cancel()
		synctest.Wait()

		// After cancel, no more callbacks.
		advance(5 * time.Second)

		assert.Equal(t, <-errCh, context.Canceled)
		assert.Equal(t, count, 2)
	})
}

func TestCorrectID(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ctx, cancel := context.WithCancel(t.Context())
		defer cancel()

		r := repeat.New()
		errCh := runRepeat(ctx, r)
		synctest.Wait()

		ch42 := make(chan int64, 1)
		ch311 := make(chan int64, 1)

		fn42 := func(ctx context.Context, id int64) bool {
			ch42 <- id
			return true
		}

		fn311 := func(ctx context.Context, id int64) bool {
			ch311 <- id
			return true
		}

		assert.NilError(t, r.Add(ctx, 42, fn42, time.Now(), time.Second))
		assert.NilError(t, r.Add(ctx, 311, fn311, time.Now(), time.Second))
		synctest.Wait()

		advance(100 * time.Millisecond)
		advance(time.Second)

		assert.Equal(t, <-ch42, int64(42))
		assert.Equal(t, <-ch311, int64(311))

		cancel()
		synctest.Wait()

		assert.Equal(t, <-errCh, context.Canceled)
	})
}

func TestReset(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ctx, cancel := context.WithCancel(t.Context())
		defer cancel()

		r := repeat.New()
		errCh := runRepeat(ctx, r)
		synctest.Wait()

		repeats, schedules, err := r.Count(ctx)
		assert.NilError(t, err)
		assert.Equal(t, repeats, 0)
		assert.Equal(t, schedules, 0)

		fn := func(ctx context.Context, id int64) bool { return true }
		assert.NilError(t, r.Add(ctx, 0, fn, time.Now(), time.Hour))

		repeats, schedules, err = r.Count(ctx)
		assert.NilError(t, err)
		assert.Equal(t, repeats, 1)
		assert.Equal(t, schedules, 0)

		assert.NilError(t, r.AddCron(ctx, 0, fn, mustParseCron("@daily")))

		repeats, schedules, err = r.Count(ctx)
		assert.NilError(t, err)
		assert.Equal(t, repeats, 1)
		assert.Equal(t, schedules, 1)

		assert.NilError(t, r.Reset(ctx))

		repeats, schedules, err = r.Count(ctx)
		assert.NilError(t, err)
		assert.Equal(t, repeats, 0)
		assert.Equal(t, schedules, 0)

		cancel()
		synctest.Wait()

		assert.Equal(t, <-errCh, context.Canceled)

		_, _, err = r.Count(ctx)
		assert.Equal(t, err, context.Canceled)
	})
}

func TestAddManyRemove(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ctx, cancel := context.WithCancel(t.Context())
		defer cancel()

		r := repeat.New()
		errCh := runRepeat(ctx, r)
		synctest.Wait()

		repeats, schedules, err := r.Count(ctx)
		assert.NilError(t, err)
		assert.Equal(t, repeats, 0)
		assert.Equal(t, schedules, 0)

		fn := func(ctx context.Context, id int64) bool { return true }
		assert.NilError(t, r.Add(ctx, 0, fn, time.Now(), time.Hour))
		assert.NilError(t, r.Add(ctx, 1, fn, time.Now(), 2*time.Hour))
		assert.NilError(t, r.AddCron(ctx, 0, fn, mustParseCron("0 * * * *")))
		assert.NilError(t, r.AddCron(ctx, 1, fn, mustParseCron("@daily")))

		assert.NilError(t, r.Remove(ctx, 1))
		assert.NilError(t, r.RemoveCron(ctx, 1))

		repeats, schedules, err = r.Count(ctx)
		assert.NilError(t, err)
		assert.Equal(t, repeats, 1)
		assert.Equal(t, schedules, 1)

		cancel()
		synctest.Wait()

		assert.Equal(t, <-errCh, context.Canceled)

		_, _, err = r.Count(ctx)
		assert.Equal(t, err, context.Canceled)
	})
}

func TestCanceled(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ctx, cancel := context.WithCancel(t.Context())
		defer cancel()

		r := repeat.New()
		errCh := runRepeat(ctx, r)
		synctest.Wait()

		cancel()
		synctest.Wait()

		assert.Equal(t, <-errCh, context.Canceled)
		assert.Equal(t, r.Add(ctx, 0, nil, time.Now(), time.Second), context.Canceled)
		assert.Equal(t, r.AddCron(ctx, 0, nil, mustParseCron("@daily")), context.Canceled)
		assert.Equal(t, r.Remove(ctx, 0), context.Canceled)
		assert.Equal(t, r.RemoveCron(ctx, 0), context.Canceled)
		assert.Equal(t, r.Reset(ctx), context.Canceled)

		_, _, err := r.Count(ctx)
		assert.Equal(t, err, context.Canceled)
	})
}

func TestRemoveNonExistent(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ctx, cancel := context.WithCancel(t.Context())
		defer cancel()

		r := repeat.New()
		errCh := runRepeat(ctx, r)
		synctest.Wait()

		assert.NilError(t, r.Remove(ctx, 0))
		assert.NilError(t, r.RemoveCron(ctx, 0))

		cancel()
		synctest.Wait()

		assert.Equal(t, <-errCh, context.Canceled)
	})
}

func TestAddTwiceFix(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ctx, cancel := context.WithCancel(t.Context())
		defer cancel()

		r := repeat.New()
		errCh := runRepeat(ctx, r)
		synctest.Wait()

		count := 0
		fn := func(ctx context.Context, id int64) bool {
			count++
			return true
		}

		assert.NilError(t, r.Add(ctx, 1, fn, time.Now(), time.Second))
		assert.NilError(t, r.Add(ctx, 0, fn, time.Now(), time.Minute))
		synctest.Wait()

		advance(100 * time.Millisecond)
		advance(time.Second)
		advance(time.Second)

		// Re-add ID 0 with an even longer interval; ID 1 should keep firing.
		assert.NilError(t, r.Add(ctx, 0, fn, time.Now(), 2*time.Minute))
		synctest.Wait()

		advance(time.Second)
		advance(time.Second)
		advance(time.Second)
		advance(100 * time.Millisecond)

		cancel()
		synctest.Wait()

		assert.Equal(t, <-errCh, context.Canceled)
		assert.Equal(t, count, 5)
	})
}

func TestAddRunOnce(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ctx, cancel := context.WithCancel(t.Context())
		defer cancel()

		r := repeat.New()
		errCh := runRepeat(ctx, r)
		synctest.Wait()

		count := 0
		fn := func(ctx context.Context, id int64) bool {
			count++
			return false
		}

		assert.NilError(t, r.Add(ctx, 0, fn, time.Now(), time.Second))
		synctest.Wait()

		advance(100 * time.Millisecond)
		for range 5 {
			advance(time.Second)
		}
		advance(100 * time.Millisecond)

		cancel()
		synctest.Wait()

		assert.Equal(t, <-errCh, context.Canceled)
		assert.Equal(t, count, 1)
	})
}
