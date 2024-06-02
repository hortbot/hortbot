package errgroupx_test

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"

	"github.com/fortytw2/leaktest"
	"github.com/hortbot/hortbot/internal/pkg/errgroupx"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"
)

type contextKey string

func TestNormal(t *testing.T) {
	defer leaktest.Check(t)()

	ctx := context.Background()
	ctx = context.WithValue(ctx, contextKey("key"), 1234)

	var done atomic.Bool

	g := errgroupx.FromContext(ctx)

	g.Go(func(ctx context.Context) error {
		v := ctx.Value(contextKey("key"))
		assert.Check(t, cmp.Equal(v, 1234))

		done.Store(true)
		return nil
	})

	assert.Check(t, g.Wait())
	assert.Check(t, done.Load())
}

func TestStop(t *testing.T) {
	defer leaktest.Check(t)()

	ctx := context.Background()

	g := errgroupx.FromContext(ctx)

	g.Go(func(ctx context.Context) error {
		<-ctx.Done()
		return nil
	})

	g.Go(func(ctx context.Context) error {
		g.Stop()
		return nil
	})

	assert.Check(t, cmp.Equal(g.Wait(), errgroupx.ErrStop))
}

func TestStopIgnored(t *testing.T) {
	defer leaktest.Check(t)()

	ctx := context.Background()

	g := errgroupx.FromContext(ctx)

	g.Go(func(ctx context.Context) error {
		<-ctx.Done()
		return nil
	})

	g.Go(func(ctx context.Context) error {
		g.Stop()
		return nil
	})

	assert.Check(t, g.WaitIgnoreStop())
}

func TestWaitIgnoreWithError(t *testing.T) {
	defer leaktest.Check(t)()

	testErr := errors.New("test error")

	ctx := context.Background()

	g := errgroupx.FromContext(ctx)

	g.Go(func(ctx context.Context) error {
		return testErr
	})

	assert.Check(t, cmp.Equal(g.WaitIgnoreStop(), testErr))
}
