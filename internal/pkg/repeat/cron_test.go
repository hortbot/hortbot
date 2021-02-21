package repeat_test

import (
	"context"
	"testing"
	"time"

	"github.com/fortytw2/leaktest"
	"github.com/hortbot/hortbot/internal/pkg/repeat"
	"github.com/robfig/cron/v3"
	"gotest.tools/v3/assert"
)

var startTime = mustParseTime("2000-10-01T03:11:00Z")

func TestParseCron(t *testing.T) {
	_, err := repeat.ParseCron("")
	assert.Error(t, err, "empty spec string")
}

func TestCronAdd(t *testing.T) {
	defer leaktest.Check(t)()

	clk := newMock(t)
	clk.Set(startTime)
	ctx, cancel := testContext(t)
	defer cancel()

	r := repeat.New(clk)
	errCh := runRepeat(ctx, r)

	ch := make(chan struct{})

	count := 0
	fn := func(ctx context.Context, id int64) bool {
		count++
		ch <- struct{}{}
		return true
	}

	assert.NilError(t, r.AddCron(ctx, 0, fn, mustParseCron("0 * * * *")))
	time.Sleep(smallDur)

	for i := 0; i < 5; i++ {
		clk.Forward(time.Hour)
		recv(t, ctx, ch)
		time.Sleep(smallDur)
	}

	cancel()

	assert.Equal(t, <-errCh, context.Canceled)
	assert.Equal(t, count, 5)
}

func TestCronAddTwice(t *testing.T) {
	defer leaktest.Check(t)()

	clk := newMock(t)
	clk.Set(startTime)
	ctx, cancel := testContext(t)
	defer cancel()

	r := repeat.New(clk)
	errCh := runRepeat(ctx, r)

	ch := make(chan struct{})

	count := 0
	fn := func(ctx context.Context, id int64) bool {
		count++
		ch <- struct{}{}
		return true
	}

	assert.NilError(t, r.AddCron(ctx, 0, fn, mustParseCron("0 * * * *")))
	time.Sleep(smallDur)

	clk.Forward(time.Hour)
	recv(t, ctx, ch)
	time.Sleep(smallDur)

	clk.Forward(time.Hour)
	recv(t, ctx, ch)
	time.Sleep(smallDur)

	runEveryDay := mustParseCron("@daily")

	assert.NilError(t, r.AddCron(ctx, 0, fn, runEveryDay))
	time.Sleep(50 * time.Millisecond)

	clk.Forward(time.Hour)
	clk.Forward(time.Hour)
	clk.Forward(time.Hour)

	time.Sleep(50 * time.Millisecond)

	cancel()

	assert.Equal(t, <-errCh, context.Canceled)
	assert.Equal(t, count, 2)
}

func TestCronAddRemove(t *testing.T) {
	defer leaktest.Check(t)()

	clk := newMock(t)
	clk.Set(startTime)
	ctx, cancel := testContext(t)
	defer cancel()

	r := repeat.New(clk)
	errCh := runRepeat(ctx, r)

	ch := make(chan struct{})

	count := 0
	fn := func(ctx context.Context, id int64) bool {
		count++
		ch <- struct{}{}
		return true
	}

	assert.NilError(t, r.AddCron(ctx, 0, fn, mustParseCron("0 * * * *")))
	time.Sleep(smallDur)

	clk.Forward(time.Hour)
	recv(t, ctx, ch)
	time.Sleep(smallDur)

	clk.Forward(time.Hour)
	recv(t, ctx, ch)
	time.Sleep(smallDur)

	assert.NilError(t, r.RemoveCron(ctx, 0))
	time.Sleep(50 * time.Millisecond)

	clk.Forward(time.Hour)
	time.Sleep(smallDur)
	clk.Forward(time.Hour)
	time.Sleep(smallDur)
	clk.Forward(time.Hour)

	time.Sleep(50 * time.Millisecond)

	cancel()

	assert.Equal(t, <-errCh, context.Canceled)
	assert.Equal(t, count, 2)
}

func TestAddCronStop(t *testing.T) {
	defer leaktest.Check(t)()

	clk := newMock(t)
	clk.Set(startTime)
	ctx, cancel := testContext(t)
	defer cancel()

	r := repeat.New(clk)
	errCh := runRepeat(ctx, r)

	ch := make(chan struct{})

	count := 0
	fn := func(ctx context.Context, id int64) bool {
		count++
		ch <- struct{}{}
		return true
	}

	assert.NilError(t, r.AddCron(ctx, 0, fn, mustParseCron("0 * * * *")))
	time.Sleep(smallDur)

	clk.Forward(time.Hour)
	recv(t, ctx, ch)
	time.Sleep(smallDur)

	clk.Forward(time.Hour)
	recv(t, ctx, ch)
	time.Sleep(smallDur)

	cancel()
	time.Sleep(50 * time.Millisecond)

	clk.Forward(time.Hour)
	time.Sleep(smallDur)
	clk.Forward(time.Hour)
	time.Sleep(smallDur)
	clk.Forward(time.Hour)

	time.Sleep(50 * time.Millisecond)

	assert.Equal(t, <-errCh, context.Canceled)
	assert.Equal(t, count, 2)
}

func TestCorrectIDCron(t *testing.T) {
	defer leaktest.Check(t)()

	clk := newMock(t)
	clk.Set(startTime)
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

	assert.NilError(t, r.AddCron(ctx, 42, fn42, mustParseCron("0 * * * *")))
	assert.NilError(t, r.AddCron(ctx, 311, fn311, mustParseCron("0 * * * *")))
	time.Sleep(50 * time.Millisecond)

	clk.Forward(time.Hour)
	time.Sleep(smallDur)
	clk.Forward(time.Second)

	time.Sleep(50 * time.Millisecond)

	assert.Equal(t, <-ch42, int64(42))
	assert.Equal(t, <-ch311, int64(311))

	cancel()

	assert.Equal(t, <-errCh, context.Canceled)
}

func TestImpossibleCron(t *testing.T) {
	defer leaktest.Check(t)()

	clk := newMock(t)
	clk.Set(startTime)
	ctx, cancel := testContext(t)
	defer cancel()

	r := repeat.New(clk)
	errCh := runRepeat(ctx, r)
	count := 0
	fn := func(ctx context.Context, id int64) bool {
		count++
		return true
	}

	assert.NilError(t, r.AddCron(ctx, 0, fn, repeat.ToCron(&cron.SpecSchedule{Month: 13, Location: time.UTC})))

	clk.Forward(time.Hour)
	time.Sleep(smallDur)
	clk.Forward(time.Hour)
	time.Sleep(smallDur)
	clk.Forward(time.Hour)
	time.Sleep(smallDur)
	clk.Forward(time.Hour)
	time.Sleep(smallDur)
	clk.Forward(time.Hour)

	time.Sleep(50 * time.Millisecond)

	cancel()

	assert.Equal(t, <-errCh, context.Canceled)
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

//nolint:golint,revive
func recv(t *testing.T, ctx context.Context, ch <-chan struct{}) {
	t.Helper()

	select {
	case <-ctx.Done():
		t.Fatal("timed out")
	case <-ch:
	}
}
