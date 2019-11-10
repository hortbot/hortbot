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

var startTime = mustParseTime("2000-10-01T03:11:00Z")

func TestParseCron(t *testing.T) {
	_, err := repeat.ParseCron("")
	assert.Error(t, err, "empty spec string")
}

func TestCronAdd(t *testing.T) {
	defer leaktest.Check(t)()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	clk := clock.NewMock()
	clk.Set(startTime)

	ch := make(chan struct{})

	count := 0
	fn := func(ctx context.Context, id int64) {
		count++
		ch <- struct{}{}
	}

	r := repeat.New(ctx, clk)

	r.AddCron(0, fn, mustParseCron("0 * * * *"))
	time.Sleep(10 * time.Millisecond)

	for i := 0; i < 5; i++ {
		clk.Forward(time.Hour)
		recv(t, ctx, ch)
		time.Sleep(10 * time.Millisecond)
	}

	r.Stop()

	assert.Equal(t, count, 5)
}

func TestCronAddTwice(t *testing.T) {
	defer leaktest.Check(t)()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	clk := clock.NewMock()
	clk.Set(startTime)

	ch := make(chan struct{})

	count := 0
	fn := func(ctx context.Context, id int64) {
		count++
		ch <- struct{}{}
	}

	r := repeat.New(ctx, clk)

	r.AddCron(0, fn, mustParseCron("0 * * * *"))
	time.Sleep(10 * time.Millisecond)

	clk.Forward(time.Hour)
	recv(t, ctx, ch)
	time.Sleep(10 * time.Millisecond)

	clk.Forward(time.Hour)
	recv(t, ctx, ch)
	time.Sleep(10 * time.Millisecond)

	runEveryDay := mustParseCron("@daily")

	r.AddCron(0, fn, runEveryDay)
	time.Sleep(50 * time.Millisecond)

	clk.Forward(time.Hour)
	clk.Forward(time.Hour)
	clk.Forward(time.Hour)

	time.Sleep(50 * time.Millisecond)

	r.Stop()

	assert.Equal(t, count, 2)
}

func TestCronAddRemove(t *testing.T) {
	defer leaktest.Check(t)()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	clk := clock.NewMock()
	clk.Set(startTime)

	ch := make(chan struct{})

	count := 0
	fn := func(ctx context.Context, id int64) {
		count++
		ch <- struct{}{}
	}

	r := repeat.New(ctx, clk)

	r.AddCron(0, fn, mustParseCron("0 * * * *"))
	time.Sleep(10 * time.Millisecond)

	clk.Forward(time.Hour)
	recv(t, ctx, ch)
	time.Sleep(10 * time.Millisecond)

	clk.Forward(time.Hour)
	recv(t, ctx, ch)
	time.Sleep(10 * time.Millisecond)

	r.RemoveCron(0)
	time.Sleep(50 * time.Millisecond)

	clk.Forward(time.Hour)
	clk.Forward(time.Hour)
	clk.Forward(time.Hour)

	time.Sleep(50 * time.Millisecond)

	r.Stop()

	assert.Equal(t, count, 2)
}

func TestAddCronStop(t *testing.T) {
	defer leaktest.Check(t)()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	clk := clock.NewMock()
	clk.Set(startTime)

	ch := make(chan struct{})

	count := 0
	fn := func(ctx context.Context, id int64) {
		count++
		ch <- struct{}{}
	}

	r := repeat.New(ctx, clk)

	r.AddCron(0, fn, mustParseCron("0 * * * *"))
	time.Sleep(10 * time.Millisecond)

	clk.Forward(time.Hour)
	recv(t, ctx, ch)
	time.Sleep(10 * time.Millisecond)

	clk.Forward(time.Hour)
	recv(t, ctx, ch)
	time.Sleep(10 * time.Millisecond)

	r.Stop()
	time.Sleep(50 * time.Millisecond)

	clk.Forward(time.Hour)
	clk.Forward(time.Hour)
	clk.Forward(time.Hour)

	time.Sleep(50 * time.Millisecond)

	assert.Equal(t, count, 2)
}

func TestCorrectIDCron(t *testing.T) {
	defer leaktest.Check(t)()

	clk := clock.NewMock()
	clk.Set(startTime)

	r := repeat.New(context.Background(), clk)

	ch42 := make(chan int64, 1)
	ch311 := make(chan int64, 1)

	fn42 := func(ctx context.Context, id int64) {
		ch42 <- id
	}

	fn311 := func(ctx context.Context, id int64) {
		ch311 <- id
	}

	r.AddCron(42, fn42, mustParseCron("0 * * * *"))
	r.AddCron(311, fn311, mustParseCron("0 * * * *"))
	time.Sleep(50 * time.Millisecond)

	clk.Forward(time.Hour)
	clk.Forward(time.Second)

	time.Sleep(50 * time.Millisecond)

	assert.Equal(t, <-ch42, int64(42))
	assert.Equal(t, <-ch311, int64(311))

	r.Stop()
}

func TestBadCron(t *testing.T) {
	t.Skip("This cron library doesn't have a constructable bad pattern.")

	defer leaktest.Check(t)()

	clk := clock.NewMock()
	clk.Set(startTime)

	count := 0
	fn := func(ctx context.Context, id int64) {
		count++
	}

	r := repeat.New(context.Background(), clk)

	r.AddCron(0, fn, nil)

	clk.Forward(time.Hour)
	clk.Forward(time.Hour)
	clk.Forward(time.Hour)
	clk.Forward(time.Hour)
	clk.Forward(time.Hour)

	time.Sleep(50 * time.Millisecond)

	r.Stop()

	assert.Equal(t, count, 0)
}

func mustParseTime(s string) time.Time {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		panic(err)
	}
	return t
}

func mustParseCron(s string) *repeat.Cron {
	e, err := repeat.ParseCron(s)
	if err != nil {
		panic(err)
	}
	return e
}

//nolint
func recv(t *testing.T, ctx context.Context, ch <-chan struct{}) {
	t.Helper()

	select {
	case <-ctx.Done():
		t.Fatal("timed out")
	case <-ch:
	}
}
