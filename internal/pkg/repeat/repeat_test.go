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
	fn := func(ctx context.Context) {
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
	fn := func(ctx context.Context) {
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
	fn := func(ctx context.Context) {
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
	fn := func(ctx context.Context) {
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
	fn := func(ctx context.Context) {
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
