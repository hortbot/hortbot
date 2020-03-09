package wqueue

import (
	"context"
	"testing"
	"time"

	"gotest.tools/v3/assert"
)

type contextKey string

func TestAttachNoDone(t *testing.T) {
	wCtx := context.Background()
	ctx := context.WithValue(wCtx, contextKey("something"), 1234)

	fn := attachFunc(wCtx)
	assert.Assert(t, fn != nil)

	wrapped, cancel := fn(ctx)
	defer cancel()
	assert.Equal(t, wrapped, ctx)
}

func TestDeadline(t *testing.T) {
	base := context.Background()
	hasCancel, cancel := context.WithCancel(base)
	defer cancel()

	now := time.Now()
	soon := now.Add(time.Minute)
	far := now.Add(time.Hour)

	hasDeadlineSoon, cancel := context.WithDeadline(base, soon)
	defer cancel()

	hasDeadlineFar, cancel := context.WithDeadline(base, far)
	defer cancel()

	test := func(inner, outer context.Context, deadline time.Time, ok bool) func(*testing.T) {
		return func(t *testing.T) {
			ctx, cancel := attachFunc(inner)(outer)
			defer cancel()

			gotDeadline, gotOk := ctx.Deadline()
			assert.Equal(t, gotDeadline, deadline)
			assert.Equal(t, gotOk, ok)
		}
	}

	t.Run("No deadline", test(base, base, time.Time{}, false))
	t.Run("Only cancel", test(base, hasCancel, time.Time{}, false))
	t.Run("Inner sooner", test(hasDeadlineSoon, hasDeadlineFar, soon, true))
	t.Run("Outer sooner", test(hasDeadlineFar, hasDeadlineSoon, soon, true))
	t.Run("Outer only", test(hasDeadlineSoon, base, soon, true))
}

func TestValue(t *testing.T) {
	base := context.WithValue(context.Background(), contextKey("base"), 1234)
	base = context.WithValue(base, contextKey("base2"), "value")
	other := context.WithValue(context.Background(), contextKey("other"), true)
	other = context.WithValue(other, contextKey("base"), 7890)
	other, cancel := context.WithCancel(other)
	defer cancel()

	ctx, cancel := attachFunc(other)(base)
	defer cancel()

	assert.Equal(t, ctx.Value(contextKey("base")), 1234)
	assert.Equal(t, ctx.Value(contextKey("base2")), "value")
	assert.Equal(t, ctx.Value(contextKey("other")), nil)
}

func TestDoneErr(t *testing.T) {
	t.Run("No cancel", func(t *testing.T) {
		a, cancelA := context.WithCancel(context.Background())
		b, cancelB := context.WithCancel(context.Background())

		ctx, cancel := attachFunc(a)(b)
		defer cancel()

		assert.NilError(t, ctx.Err())
		cancelA()
		cancelB()
	})

	t.Run("Inner canceled", func(t *testing.T) {
		a, cancelA := context.WithCancel(context.Background())
		b, cancelB := context.WithCancel(context.Background())

		ctx, cancel := attachFunc(a)(b)
		defer cancel()

		done := ctx.Done()
		cancelA()

		<-done
		assert.Equal(t, ctx.Err(), context.Canceled)
		cancelB()
	})

	t.Run("Outer canceled", func(t *testing.T) {
		a, cancelA := context.WithCancel(context.Background())
		b, cancelB := context.WithCancel(context.Background())

		ctx, cancel := attachFunc(a)(b)
		defer cancel()

		done := ctx.Done()
		cancelB()

		<-done
		assert.Equal(t, ctx.Err(), context.Canceled)
		cancelA()
	})
}
