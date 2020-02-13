package repeat_test

import (
	"context"
	"testing"
	"time"

	"github.com/fortytw2/leaktest"
	"github.com/hortbot/hortbot/internal/pkg/ctxlog"
	"github.com/hortbot/hortbot/internal/pkg/repeat"
	"github.com/hortbot/hortbot/internal/pkg/testutil"
	"github.com/leononame/clock"
	"gotest.tools/v3/assert"
)

const smallDur = 15 * time.Millisecond

func newMock(t testing.TB) *clock.Mock {
	t.Helper()
	clk := clock.NewMock()
	ts, err := time.Parse(time.RFC3339, "2000-01-01T00:00:00Z")
	assert.NilError(t, err)
	clk.Set(ts)
	return clk
}

func testContext(t testing.TB) (context.Context, context.CancelFunc) {
	logger := testutil.Logger(t)
	ctx := ctxlog.WithLogger(context.Background(), logger)
	return context.WithTimeout(ctx, 5*time.Second)
}

func runRepeat(ctx context.Context, r *repeat.Repeater) <-chan error {
	errCh := make(chan error, 1)
	go func() {
		errCh <- r.Run(ctx)
	}()
	return errCh
}

func TestDoNothing(t *testing.T) {
	defer leaktest.Check(t)()

	clk := newMock(t)
	ctx, cancel := testContext(t)
	defer cancel()

	r := repeat.New(clk)
	errCh := runRepeat(ctx, r)

	cancel()

	assert.Equal(t, <-errCh, context.Canceled)
}

func TestAdd(t *testing.T) {
	defer leaktest.Check(t)()

	clk := newMock(t)
	ctx, cancel := testContext(t)
	defer cancel()

	r := repeat.New(clk)
	errCh := runRepeat(ctx, r)

	count := 0
	fn := func(ctx context.Context, id int64) bool {
		count++
		return true
	}

	assert.NilError(t, r.Add(ctx, 0, fn, clk.Now(), time.Second))

	clk.Forward(100 * time.Millisecond)
	time.Sleep(smallDur)
	clk.Forward(time.Second)
	time.Sleep(smallDur)
	clk.Forward(time.Second)
	time.Sleep(smallDur)
	clk.Forward(time.Second)
	time.Sleep(smallDur)
	clk.Forward(time.Second)
	time.Sleep(smallDur)
	clk.Forward(time.Second)
	time.Sleep(smallDur)
	clk.Forward(100 * time.Millisecond)

	time.Sleep(50 * time.Millisecond)

	cancel()

	assert.Equal(t, <-errCh, context.Canceled)
	assert.Equal(t, count, 5)
}

func TestAddWithInit(t *testing.T) {
	defer leaktest.Check(t)()

	clk := newMock(t)
	ctx, cancel := testContext(t)
	defer cancel()

	r := repeat.New(clk)
	errCh := runRepeat(ctx, r)

	count := 0
	fn := func(ctx context.Context, id int64) bool {
		count++
		return true
	}

	assert.NilError(t, r.Add(ctx, 0, fn, clk.Now().Add(-time.Second/2), time.Second))

	clk.Forward(100 * time.Millisecond)
	time.Sleep(smallDur)
	clk.Forward(time.Second)
	time.Sleep(50 * time.Millisecond)

	clk.Forward(time.Second)
	time.Sleep(smallDur)
	clk.Forward(time.Second)
	time.Sleep(smallDur)
	clk.Forward(time.Second)
	time.Sleep(smallDur)
	clk.Forward(time.Second)
	clk.Forward(100 * time.Millisecond)

	time.Sleep(50 * time.Millisecond)

	cancel()

	assert.Equal(t, <-errCh, context.Canceled)
	assert.Equal(t, count, 5)
}

func TestAddWithInitCancel(t *testing.T) {
	defer leaktest.Check(t)()

	clk := newMock(t)
	ctx, cancel := testContext(t)
	defer cancel()

	r := repeat.New(clk)
	errCh := runRepeat(ctx, r)

	count := 0
	fn := func(ctx context.Context, id int64) bool {
		count++
		return true
	}

	assert.NilError(t, r.Add(ctx, 0, fn, clk.Now().Add(-time.Second/2), time.Second))

	time.Sleep(50 * time.Millisecond)
	cancel()
	time.Sleep(50 * time.Millisecond)

	clk.Forward(100 * time.Millisecond)
	time.Sleep(smallDur)
	clk.Forward(time.Second)
	time.Sleep(50 * time.Millisecond)

	clk.Forward(time.Second)
	time.Sleep(smallDur)
	clk.Forward(time.Second)
	time.Sleep(smallDur)
	clk.Forward(time.Second)
	time.Sleep(smallDur)
	clk.Forward(time.Second)
	time.Sleep(smallDur)
	clk.Forward(100 * time.Millisecond)

	time.Sleep(50 * time.Millisecond)

	assert.Equal(t, <-errCh, context.Canceled)
	assert.Equal(t, count, 0)
}

func TestAddTwice(t *testing.T) {
	defer leaktest.Check(t)()

	clk := newMock(t)
	ctx, cancel := testContext(t)
	defer cancel()

	r := repeat.New(clk)
	errCh := runRepeat(ctx, r)

	count := 0
	fn := func(ctx context.Context, id int64) bool {
		count++
		return true
	}

	assert.NilError(t, r.Add(ctx, 0, fn, clk.Now(), time.Second))

	clk.Forward(100 * time.Millisecond)
	time.Sleep(smallDur)
	clk.Forward(time.Second)
	time.Sleep(smallDur)
	clk.Forward(time.Second)
	time.Sleep(50 * time.Millisecond)

	assert.NilError(t, r.Add(ctx, 0, fn, clk.Now(), time.Minute))
	time.Sleep(50 * time.Millisecond)

	clk.Forward(time.Second)
	time.Sleep(smallDur)
	clk.Forward(time.Second)
	time.Sleep(smallDur)
	clk.Forward(time.Second)
	time.Sleep(smallDur)
	clk.Forward(100 * time.Millisecond)

	time.Sleep(50 * time.Millisecond)

	cancel()

	assert.Equal(t, <-errCh, context.Canceled)
	assert.Equal(t, count, 2)
}

func TestAddRemove(t *testing.T) {
	defer leaktest.Check(t)()

	clk := newMock(t)
	ctx, cancel := testContext(t)
	defer cancel()

	r := repeat.New(clk)
	errCh := runRepeat(ctx, r)

	count := 0
	fn := func(ctx context.Context, id int64) bool {
		count++
		return true
	}

	assert.NilError(t, r.Add(ctx, 0, fn, clk.Now(), time.Second))

	clk.Forward(100 * time.Millisecond)
	time.Sleep(smallDur)
	clk.Forward(time.Second)
	time.Sleep(smallDur)
	clk.Forward(time.Second)
	time.Sleep(50 * time.Millisecond)

	assert.NilError(t, r.Remove(ctx, 0))
	time.Sleep(50 * time.Millisecond)

	clk.Forward(time.Second)
	time.Sleep(smallDur)
	clk.Forward(time.Second)
	time.Sleep(smallDur)
	clk.Forward(time.Second)
	time.Sleep(smallDur)
	clk.Forward(100 * time.Millisecond)

	time.Sleep(50 * time.Millisecond)

	cancel()

	assert.Equal(t, <-errCh, context.Canceled)
	assert.Equal(t, count, 2)
}

func TestAddStop(t *testing.T) {
	defer leaktest.Check(t)()

	clk := newMock(t)
	ctx, cancel := testContext(t)
	defer cancel()

	r := repeat.New(clk)
	errCh := runRepeat(ctx, r)

	count := 0
	fn := func(ctx context.Context, id int64) bool {
		count++
		return true
	}

	assert.NilError(t, r.Add(ctx, 0, fn, clk.Now(), time.Second))

	clk.Forward(100 * time.Millisecond)
	time.Sleep(smallDur)
	clk.Forward(time.Second)
	time.Sleep(smallDur)
	clk.Forward(time.Second)
	time.Sleep(50 * time.Millisecond)

	cancel()
	time.Sleep(50 * time.Millisecond)

	clk.Forward(time.Second)
	time.Sleep(smallDur)
	clk.Forward(time.Second)
	time.Sleep(smallDur)
	clk.Forward(time.Second)
	time.Sleep(smallDur)
	clk.Forward(100 * time.Millisecond)

	time.Sleep(50 * time.Millisecond)

	assert.Equal(t, <-errCh, context.Canceled)
	assert.Equal(t, count, 2)
}

func TestCorrectID(t *testing.T) {
	defer leaktest.Check(t)()

	clk := newMock(t)
	ctx, cancel := testContext(t)
	defer cancel()

	r := repeat.New(clk)
	errCh := runRepeat(ctx, r)

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

	assert.NilError(t, r.Add(ctx, 42, fn42, clk.Now(), time.Second))
	assert.NilError(t, r.Add(ctx, 311, fn311, clk.Now(), time.Second))

	clk.Forward(100 * time.Millisecond)
	time.Sleep(smallDur)
	clk.Forward(time.Second)

	time.Sleep(50 * time.Millisecond)

	assert.Equal(t, <-ch42, int64(42))
	assert.Equal(t, <-ch311, int64(311))

	cancel()

	assert.Equal(t, <-errCh, context.Canceled)
}

func TestReset(t *testing.T) {
	defer leaktest.Check(t)()

	clk := newMock(t)
	ctx, cancel := testContext(t)
	defer cancel()

	r := repeat.New(clk)
	errCh := runRepeat(ctx, r)

	repeats, schedules, err := r.Count(ctx)
	assert.NilError(t, err)
	assert.Equal(t, repeats, 0)
	assert.Equal(t, schedules, 0)

	fn := func(ctx context.Context, id int64) bool { return true }
	assert.NilError(t, r.Add(ctx, 0, fn, clk.Now(), time.Hour))

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

	assert.Equal(t, <-errCh, context.Canceled)

	_, _, err = r.Count(ctx)
	assert.Equal(t, err, context.Canceled)
}

func TestAddManyRemove(t *testing.T) {
	defer leaktest.Check(t)()

	clk := newMock(t)
	ctx, cancel := testContext(t)
	defer cancel()

	r := repeat.New(clk)
	errCh := runRepeat(ctx, r)

	repeats, schedules, err := r.Count(ctx)
	assert.NilError(t, err)
	assert.Equal(t, repeats, 0)
	assert.Equal(t, schedules, 0)

	fn := func(ctx context.Context, id int64) bool { return true }
	assert.NilError(t, r.Add(ctx, 0, fn, clk.Now(), time.Hour))
	assert.NilError(t, r.Add(ctx, 1, fn, clk.Now(), 2*time.Hour))
	assert.NilError(t, r.AddCron(ctx, 0, fn, mustParseCron("0 * * * *")))
	assert.NilError(t, r.AddCron(ctx, 1, fn, mustParseCron("@daily")))

	assert.NilError(t, r.Remove(ctx, 1))
	assert.NilError(t, r.RemoveCron(ctx, 1))

	repeats, schedules, err = r.Count(ctx)
	assert.NilError(t, err)
	assert.Equal(t, repeats, 1)
	assert.Equal(t, schedules, 1)

	cancel()

	assert.Equal(t, <-errCh, context.Canceled)

	_, _, err = r.Count(ctx)
	assert.Equal(t, err, context.Canceled)
}

func TestCanceled(t *testing.T) {
	defer leaktest.Check(t)()

	clk := newMock(t)
	ctx, cancel := testContext(t)
	defer cancel()

	r := repeat.New(clk)
	errCh := runRepeat(ctx, r)

	cancel()

	assert.Equal(t, <-errCh, context.Canceled)
	assert.Equal(t, r.Add(ctx, 0, nil, clk.Now(), time.Second), context.Canceled)
	assert.Equal(t, r.AddCron(ctx, 0, nil, mustParseCron("@daily")), context.Canceled)
	assert.Equal(t, r.Remove(ctx, 0), context.Canceled)
	assert.Equal(t, r.RemoveCron(ctx, 0), context.Canceled)
	assert.Equal(t, r.Reset(ctx), context.Canceled)

	_, _, err := r.Count(ctx)
	assert.Equal(t, err, context.Canceled)
}

func TestRemoveNonExistent(t *testing.T) {
	defer leaktest.Check(t)()

	clk := newMock(t)
	ctx, cancel := testContext(t)
	defer cancel()

	r := repeat.New(clk)
	errCh := runRepeat(ctx, r)

	assert.NilError(t, r.Remove(ctx, 0))
	assert.NilError(t, r.RemoveCron(ctx, 0))

	cancel()

	assert.Equal(t, <-errCh, context.Canceled)
}

func TestAddTwiceFix(t *testing.T) {
	defer leaktest.Check(t)()

	clk := newMock(t)
	ctx, cancel := testContext(t)
	defer cancel()

	r := repeat.New(clk)
	errCh := runRepeat(ctx, r)

	count := 0
	fn := func(ctx context.Context, id int64) bool {
		count++
		return true
	}

	assert.NilError(t, r.Add(ctx, 1, fn, clk.Now(), time.Second))
	assert.NilError(t, r.Add(ctx, 0, fn, clk.Now(), time.Minute))

	clk.Forward(100 * time.Millisecond)
	time.Sleep(smallDur)
	clk.Forward(time.Second)
	time.Sleep(smallDur)
	clk.Forward(time.Second)
	time.Sleep(50 * time.Millisecond)

	assert.NilError(t, r.Add(ctx, 0, fn, clk.Now(), 2*time.Minute))
	time.Sleep(50 * time.Millisecond)

	clk.Forward(time.Second)
	time.Sleep(smallDur)
	clk.Forward(time.Second)
	time.Sleep(smallDur)
	clk.Forward(time.Second)
	time.Sleep(smallDur)
	clk.Forward(100 * time.Millisecond)

	time.Sleep(50 * time.Millisecond)

	cancel()

	assert.Equal(t, <-errCh, context.Canceled)
	assert.Equal(t, count, 5)
}

func TestAddRunOnce(t *testing.T) {
	defer leaktest.Check(t)()

	clk := newMock(t)
	ctx, cancel := testContext(t)
	defer cancel()

	r := repeat.New(clk)
	errCh := runRepeat(ctx, r)

	count := 0
	fn := func(ctx context.Context, id int64) bool {
		count++
		return false
	}

	assert.NilError(t, r.Add(ctx, 0, fn, clk.Now(), time.Second))

	clk.Forward(100 * time.Millisecond)
	time.Sleep(smallDur)
	clk.Forward(time.Second)
	time.Sleep(smallDur)
	clk.Forward(time.Second)
	time.Sleep(smallDur)
	clk.Forward(time.Second)
	time.Sleep(smallDur)
	clk.Forward(time.Second)
	time.Sleep(smallDur)
	clk.Forward(time.Second)
	time.Sleep(smallDur)
	clk.Forward(100 * time.Millisecond)

	time.Sleep(50 * time.Millisecond)

	cancel()

	assert.Equal(t, <-errCh, context.Canceled)
	assert.Equal(t, count, 1)
}
