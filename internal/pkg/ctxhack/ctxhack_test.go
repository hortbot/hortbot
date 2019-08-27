package ctxhack

import (
	"context"
	"testing"

	"gotest.tools/v3/assert"
)

type contextKey string

const key contextKey = "key"

func TestCombineValues(t *testing.T) {
	background := context.Background()
	ctx1 := context.WithValue(background, key, 1)
	ctx2 := context.WithValue(background, key, 2)
	ctx3 := context.WithValue(background, key, 3)

	ctx := Combine(ctx1, ctx2, ctx3)

	assert.Equal(t, ctx.Value(key), 2)
	assert.Equal(t, ctx.Value(1234), nil)
}

func TestCombineCancel(t *testing.T) {
	background := context.Background()
	ctx1 := context.WithValue(background, key, 1)
	ctx2 := context.WithValue(background, key, 2)
	ctx3 := context.WithValue(background, key, 3)

	ctx1, cancel := context.WithCancel(ctx1)
	cancel()

	ctx := Combine(ctx1, ctx2, ctx3)

	assert.Equal(t, ctx.Err(), context.Canceled)
}

func TestCombineCancelValue(t *testing.T) {
	background := context.Background()
	ctx1 := context.WithValue(background, key, 1)
	ctx2 := context.WithValue(background, key, 2)
	ctx3 := context.WithValue(background, key, 3)

	ctx2, cancel := context.WithCancel(ctx2)
	cancel()

	ctx := Combine(ctx1, ctx2, ctx3)

	assert.Equal(t, ctx.Err(), nil)
}
