package repeat_test

import (
	"context"
	"testing"
	"time"

	"github.com/efritz/glock"
	"github.com/fortytw2/leaktest"
	"github.com/hortbot/hortbot/internal/pkg/repeat"
	"gotest.tools/assert"
)

func TestDoNothing(t *testing.T) {
	defer leaktest.Check(t)()

	clk := glock.NewMockClock()

	r := repeat.New(context.Background(), clk)
	r.Stop()
}

func TestNilContext(t *testing.T) {
	defer leaktest.Check(t)()

	clk := glock.NewMockClock()

	r := repeat.New(nil, clk)
	r.Stop()
}

func TestAdd(t *testing.T) {
	defer leaktest.Check(t)()

	clk := glock.NewMockClock()

	r := repeat.New(context.Background(), clk)

	count := 0
	fn := func(ctx context.Context, id int64) {
		count++
	}

	r.Add(0, fn, time.Second, 0)

	clk.Advance(100 * time.Millisecond)
	clk.Advance(time.Second)
	clk.Advance(time.Second)
	clk.Advance(time.Second)
	clk.Advance(time.Second)
	clk.Advance(time.Second)
	clk.Advance(100 * time.Millisecond)

	time.Sleep(10 * time.Millisecond)

	r.Stop()

	assert.Equal(t, count, 5)
}

func TestAddWithInit(t *testing.T) {
	defer leaktest.Check(t)()

	clk := glock.NewMockClock()

	r := repeat.New(context.Background(), clk)

	count := 0
	fn := func(ctx context.Context, id int64) {
		count++
	}

	r.Add(0, fn, time.Second, time.Second)

	clk.Advance(100 * time.Millisecond)
	clk.Advance(time.Second)
	time.Sleep(10 * time.Millisecond)

	clk.Advance(time.Second)
	clk.Advance(time.Second)
	clk.Advance(time.Second)
	clk.Advance(time.Second)
	clk.Advance(100 * time.Millisecond)

	time.Sleep(10 * time.Millisecond)

	r.Stop()

	assert.Equal(t, count, 4)
}

func TestAddWithInitCancel(t *testing.T) {
	defer leaktest.Check(t)()

	clk := glock.NewMockClock()

	ctx, cancel := context.WithCancel(context.Background())
	r := repeat.New(ctx, clk)

	count := 0
	fn := func(ctx context.Context, id int64) {
		count++
	}

	r.Add(0, fn, time.Second, time.Second)
	cancel()
	time.Sleep(10 * time.Millisecond)

	clk.Advance(100 * time.Millisecond)
	clk.Advance(time.Second)
	time.Sleep(10 * time.Millisecond)

	clk.Advance(time.Second)
	clk.Advance(time.Second)
	clk.Advance(time.Second)
	clk.Advance(time.Second)
	clk.Advance(100 * time.Millisecond)

	time.Sleep(10 * time.Millisecond)

	r.Stop()

	assert.Equal(t, count, 0)
}

func TestAddTwice(t *testing.T) {
	defer leaktest.Check(t)()

	clk := glock.NewMockClock()

	r := repeat.New(context.Background(), clk)

	count := 0
	fn := func(ctx context.Context, id int64) {
		count++
	}

	r.Add(0, fn, time.Second, 0)

	clk.Advance(100 * time.Millisecond)
	clk.Advance(time.Second)
	clk.Advance(time.Second)
	time.Sleep(10 * time.Millisecond)

	r.Add(0, fn, time.Minute, 0)
	time.Sleep(10 * time.Millisecond)

	clk.Advance(time.Second)
	clk.Advance(time.Second)
	clk.Advance(time.Second)
	clk.Advance(100 * time.Millisecond)

	time.Sleep(10 * time.Millisecond)

	r.Stop()

	assert.Equal(t, count, 2)
}

func TestAddRemove(t *testing.T) {
	defer leaktest.Check(t)()

	clk := glock.NewMockClock()

	r := repeat.New(context.Background(), clk)

	count := 0
	fn := func(ctx context.Context, id int64) {
		count++
	}

	r.Add(0, fn, time.Second, 0)

	clk.Advance(100 * time.Millisecond)
	clk.Advance(time.Second)
	clk.Advance(time.Second)
	time.Sleep(10 * time.Millisecond)

	r.Remove(0)
	time.Sleep(10 * time.Millisecond)

	clk.Advance(time.Second)
	clk.Advance(time.Second)
	clk.Advance(time.Second)
	clk.Advance(100 * time.Millisecond)

	time.Sleep(10 * time.Millisecond)

	r.Stop()

	assert.Equal(t, count, 2)
}

func TestCorrectID(t *testing.T) {
	defer leaktest.Check(t)()

	clk := glock.NewMockClock()

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

	clk.Advance(100 * time.Millisecond)
	clk.Advance(time.Second)

	time.Sleep(10 * time.Millisecond)

	r.Stop()

	assert.Equal(t, <-ch42, int64(42))
	assert.Equal(t, <-ch311, int64(311))
}
