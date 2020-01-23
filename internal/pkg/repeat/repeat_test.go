package repeat_test

import (
	"context"
	"testing"
	"time"

	"github.com/fortytw2/leaktest"
	"github.com/hortbot/hortbot/internal/pkg/repeat"
	"github.com/leononame/clock"
	"gotest.tools/v3/assert"
)

func TestDoNothing(t *testing.T) {
	defer leaktest.Check(t)()

	clk := clock.NewMock()

	r := repeat.New(context.Background(), clk)
	r.Stop()
}

func TestAdd(t *testing.T) {
	defer leaktest.Check(t)()

	clk := clock.NewMock()

	r := repeat.New(context.Background(), clk)

	count := 0
	fn := func(ctx context.Context, id int64) {
		count++
	}

	r.Add(0, fn, time.Second, 0)

	clk.Forward(100 * time.Millisecond)
	clk.Forward(time.Second)
	clk.Forward(time.Second)
	clk.Forward(time.Second)
	clk.Forward(time.Second)
	clk.Forward(time.Second)
	clk.Forward(100 * time.Millisecond)

	time.Sleep(50 * time.Millisecond)

	r.Stop()

	assert.Equal(t, count, 5)
}

func TestAddWithInit(t *testing.T) {
	defer leaktest.Check(t)()

	clk := clock.NewMock()

	r := repeat.New(context.Background(), clk)

	count := 0
	fn := func(ctx context.Context, id int64) {
		count++
	}

	r.Add(0, fn, time.Second, time.Second)

	clk.Forward(100 * time.Millisecond)
	clk.Forward(time.Second)
	time.Sleep(50 * time.Millisecond)

	clk.Forward(time.Second)
	clk.Forward(time.Second)
	clk.Forward(time.Second)
	clk.Forward(time.Second)
	clk.Forward(100 * time.Millisecond)

	time.Sleep(50 * time.Millisecond)

	r.Stop()

	assert.Equal(t, count, 5)
}

func TestAddWithInitCancel(t *testing.T) {
	defer leaktest.Check(t)()

	clk := clock.NewMock()

	ctx, cancel := context.WithCancel(context.Background())
	r := repeat.New(ctx, clk)

	count := 0
	fn := func(ctx context.Context, id int64) {
		count++
	}

	r.Add(0, fn, time.Second, time.Second)
	cancel()
	time.Sleep(50 * time.Millisecond)

	clk.Forward(100 * time.Millisecond)
	clk.Forward(time.Second)
	time.Sleep(50 * time.Millisecond)

	clk.Forward(time.Second)
	clk.Forward(time.Second)
	clk.Forward(time.Second)
	clk.Forward(time.Second)
	clk.Forward(100 * time.Millisecond)

	time.Sleep(50 * time.Millisecond)

	r.Stop()

	assert.Equal(t, count, 0)
}

func TestAddTwice(t *testing.T) {
	defer leaktest.Check(t)()

	clk := clock.NewMock()

	r := repeat.New(context.Background(), clk)

	count := 0
	fn := func(ctx context.Context, id int64) {
		count++
	}

	r.Add(0, fn, time.Second, 0)

	clk.Forward(100 * time.Millisecond)
	clk.Forward(time.Second)
	clk.Forward(time.Second)
	time.Sleep(50 * time.Millisecond)

	r.Add(0, fn, time.Minute, 0)
	time.Sleep(50 * time.Millisecond)

	clk.Forward(time.Second)
	clk.Forward(time.Second)
	clk.Forward(time.Second)
	clk.Forward(100 * time.Millisecond)

	time.Sleep(50 * time.Millisecond)

	r.Stop()

	assert.Equal(t, count, 2)
}

func TestAddRemove(t *testing.T) {
	defer leaktest.Check(t)()

	clk := clock.NewMock()

	r := repeat.New(context.Background(), clk)

	count := 0
	fn := func(ctx context.Context, id int64) {
		count++
	}

	r.Add(0, fn, time.Second, 0)

	clk.Forward(100 * time.Millisecond)
	clk.Forward(time.Second)
	clk.Forward(time.Second)
	time.Sleep(50 * time.Millisecond)

	r.Remove(0)
	time.Sleep(50 * time.Millisecond)

	clk.Forward(time.Second)
	clk.Forward(time.Second)
	clk.Forward(time.Second)
	clk.Forward(100 * time.Millisecond)

	time.Sleep(50 * time.Millisecond)

	r.Stop()

	assert.Equal(t, count, 2)
}

func TestAddStop(t *testing.T) {
	defer leaktest.Check(t)()

	clk := clock.NewMock()

	r := repeat.New(context.Background(), clk)

	count := 0
	fn := func(ctx context.Context, id int64) {
		count++
	}

	r.Add(0, fn, time.Second, 0)

	clk.Forward(100 * time.Millisecond)
	clk.Forward(time.Second)
	clk.Forward(time.Second)
	time.Sleep(50 * time.Millisecond)

	r.Stop()
	time.Sleep(50 * time.Millisecond)

	clk.Forward(time.Second)
	clk.Forward(time.Second)
	clk.Forward(time.Second)
	clk.Forward(100 * time.Millisecond)

	time.Sleep(50 * time.Millisecond)

	assert.Equal(t, count, 2)
}

func TestCorrectID(t *testing.T) {
	defer leaktest.Check(t)()

	clk := clock.NewMock()

	r := repeat.New(context.Background(), clk)

	ch42 := make(chan int64, 1)
	ch311 := make(chan int64, 1)

	fn42 := func(ctx context.Context, id int64) {
		ch42 <- id
	}

	fn311 := func(ctx context.Context, id int64) {
		ch311 <- id
	}

	r.Add(42, fn42, time.Second, 0)
	r.Add(311, fn311, time.Second, 0)

	clk.Forward(100 * time.Millisecond)
	clk.Forward(time.Second)

	time.Sleep(50 * time.Millisecond)

	assert.Equal(t, <-ch42, int64(42))
	assert.Equal(t, <-ch311, int64(311))

	r.Stop()
}

func TestReset(t *testing.T) {
	defer leaktest.Check(t)()

	clk := clock.NewMock()

	r := repeat.New(context.Background(), clk)

	repeats, schedules := r.Count()
	assert.Equal(t, repeats, 0)
	assert.Equal(t, schedules, 0)

	fn := func(ctx context.Context, id int64) {}
	r.Add(0, fn, time.Second, 0)

	repeats, schedules = r.Count()
	assert.Equal(t, repeats, 1)
	assert.Equal(t, schedules, 0)

	r.AddCron(0, fn, mustParseCron("0 * * * *"))

	repeats, schedules = r.Count()
	assert.Equal(t, repeats, 1)
	assert.Equal(t, schedules, 1)

	r.Reset()

	repeats, schedules = r.Count()
	assert.Equal(t, repeats, 0)
	assert.Equal(t, schedules, 0)
}
