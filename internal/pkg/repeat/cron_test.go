//nolint:paralleltest
package repeat_test

import (
	"context"
	"testing"
	"testing/synctest"
	"time"

	"github.com/hortbot/hortbot/internal/pkg/must"
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
	synctest.Test(t, func(t *testing.T) {
		advanceToStartTime(t)

		ctx, cancel := context.WithCancel(t.Context())
		defer cancel()

		r := repeat.New()
		errCh := runRepeat(ctx, r)
		synctest.Wait()

		ch := make(chan struct{})

		count := 0
		fn := func(ctx context.Context, id int64) bool {
			count++
			ch <- struct{}{}
			return true
		}

		assert.NilError(t, r.AddCron(ctx, 0, fn, mustParseCron("0 * * * *")))
		synctest.Wait()

		for range 5 {
			time.Sleep(time.Hour)
			recv(t, ctx, ch)
			synctest.Wait()
		}

		cancel()
		synctest.Wait()

		assert.Equal(t, <-errCh, context.Canceled)
		assert.Equal(t, count, 5)
	})
}

func TestCronAddTwice(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		advanceToStartTime(t)

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

		assert.NilError(t, r.AddCron(ctx, 0, fn, mustParseCron("0 * * * *")))
		synctest.Wait()

		advance(time.Hour)

		advance(time.Hour)

		assert.Equal(t, count, 2)

		// Replace hourly with noon-daily (UTC); noon UTC is far enough from
		// startTime (03:11 UTC) that advancing 3h won't cross noon.
		// If startTime changes, this schedule may need updating.
		runAtNoon := mustParseCron("CRON_TZ=UTC 0 12 * * *")

		assert.NilError(t, r.AddCron(ctx, 0, fn, runAtNoon))
		synctest.Wait()

		for range 3 {
			advance(time.Hour)
		}

		assert.Equal(t, count, 2)

		cancel()
		synctest.Wait()

		assert.Equal(t, <-errCh, context.Canceled)
	})
}

func TestCronAddRemove(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		advanceToStartTime(t)

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

		assert.NilError(t, r.AddCron(ctx, 0, fn, mustParseCron("0 * * * *")))
		synctest.Wait()

		advance(time.Hour)

		advance(time.Hour)

		assert.Equal(t, count, 2)

		assert.NilError(t, r.RemoveCron(ctx, 0))
		synctest.Wait()

		for range 3 {
			advance(time.Hour)
		}

		assert.Equal(t, count, 2)

		cancel()
		synctest.Wait()

		assert.Equal(t, <-errCh, context.Canceled)
	})
}

func TestAddCronStop(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		advanceToStartTime(t)

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

		assert.NilError(t, r.AddCron(ctx, 0, fn, mustParseCron("0 * * * *")))
		synctest.Wait()

		advance(time.Hour)

		advance(time.Hour)

		assert.Equal(t, count, 2)

		cancel()
		synctest.Wait()

		// After cancel, no more callbacks.
		advance(5 * time.Hour)

		assert.Equal(t, <-errCh, context.Canceled)
		assert.Equal(t, count, 2)
	})
}

func TestCorrectIDCron(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		advanceToStartTime(t)

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

		assert.NilError(t, r.AddCron(ctx, 42, fn42, mustParseCron("0 * * * *")))
		assert.NilError(t, r.AddCron(ctx, 311, fn311, mustParseCron("0 * * * *")))
		synctest.Wait()

		advance(time.Hour)
		advance(time.Second)

		assert.Equal(t, <-ch42, int64(42))
		assert.Equal(t, <-ch311, int64(311))

		cancel()
		synctest.Wait()

		assert.Equal(t, <-errCh, context.Canceled)
	})
}

func TestImpossibleCron(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		advanceToStartTime(t)

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

		assert.NilError(t, r.AddCron(ctx, 0, fn, repeat.ToCron(&cron.SpecSchedule{Month: 13, Location: time.UTC})))
		synctest.Wait()

		for range 5 {
			advance(time.Hour)
		}

		cancel()
		synctest.Wait()

		assert.Equal(t, <-errCh, context.Canceled)
		assert.Equal(t, count, 0)
	})
}

// advanceToStartTime advances the synctest fake clock from its initial
// time (2000-01-01T00:00:00Z) to startTime (2000-10-01T03:11:00Z).
//
// This must be called before starting any goroutines with pending timers
// (e.g. before runRepeat), otherwise the bubble will tick through every
// intermediate timer event during the sleep.
func advanceToStartTime(t *testing.T) {
	t.Helper()
	// synctest starts at 2000-01-01T00:00:00Z
	synctestEpoch := time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
	diff := startTime.Sub(synctestEpoch)
	if diff > 0 {
		time.Sleep(diff)
	}
	if now := time.Now(); !now.Equal(startTime) {
		t.Fatalf("advanceToStartTime: expected %v, got %v", startTime, now)
	}
}

func mustParseTime(s string) time.Time {
	return must.Must(time.Parse(time.RFC3339, s))
}

func mustParseCron(s string) *repeat.Cron {
	return must.Must(repeat.ParseCron(s))
}

func recv(t *testing.T, ctx context.Context, ch <-chan struct{}) {
	t.Helper()

	select {
	case <-ctx.Done():
		t.Fatal("timed out")
	case <-ch:
	}
}
