package botstate_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"gotest.tools/v3/assert"
)

const concurrentGoroutines = 8

func TestCheckAndMarkCooldownConcurrent(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
	defer cancel()

	db, _ := freshStore(t)

	results, errs := runConcurrent(ctx, concurrentGoroutines, func(_ int) (bool, error) {
		return db.CheckAndMarkCooldown(ctx, "channel", "key", time.Hour)
	})

	winners := 0
	for i, err := range errs {
		assert.NilError(t, err, "goroutine %d", i)
		if !results[i] {
			winners++
		}
	}
	assert.Equal(t, winners, 1, "exactly one caller should observe the missing cooldown")
}

func TestConfirmConcurrent(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
	defer cancel()

	db, _ := freshStore(t)

	results, errs := runConcurrent(ctx, concurrentGoroutines, func(_ int) (bool, error) {
		return db.Confirm(ctx, "channel", "user", "key", time.Hour)
	})

	creates, deletes := 0, 0
	for i, err := range errs {
		assert.NilError(t, err, "goroutine %d", i)
		if results[i] {
			deletes++
		} else {
			creates++
		}
	}
	diff := creates - deletes
	assert.Assert(t, diff == 0 || diff == 1,
		"expected creates-deletes to be 0 or 1, got %d (creates=%d deletes=%d)",
		diff, creates, deletes)
}

func runConcurrent(_ context.Context, n int, fn func(i int) (bool, error)) (results []bool, errs []error) {
	results = make([]bool, n)
	errs = make([]error, n)

	var wg sync.WaitGroup
	start := make(chan struct{})
	for i := range n {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-start
			results[i], errs[i] = fn(i)
		}(i)
	}
	close(start)
	wg.Wait()
	return results, errs
}
