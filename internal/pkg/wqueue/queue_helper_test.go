package wqueue

import (
	"context"
	"testing"

	"gotest.tools/v3/assert"
)

func TestGetState2(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())

	a := make(chan int, 1)
	b := make(chan int, 1)

	a <- 1

	v, err := getState2(ctx, a, b)
	assert.Equal(t, v, 1)
	assert.NilError(t, err)

	b <- 2
	v, err = getState2(ctx, a, b)
	assert.Equal(t, v, 2)
	assert.NilError(t, err)

	cancel()

	v, err = getState2(ctx, a, b)
	assert.Equal(t, v, 0)
	assert.Equal(t, err, context.Canceled)
}

func TestGetState4(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())

	a := make(chan int, 1)
	b := make(chan int, 1)
	c := make(chan int, 1)
	d := make(chan int, 1)

	a <- 1

	v, err := getState4(ctx, a, b, c, d)
	assert.Equal(t, v, 1)
	assert.NilError(t, err)

	b <- 2
	v, err = getState4(ctx, a, b, c, d)
	assert.Equal(t, v, 2)
	assert.NilError(t, err)

	c <- 3
	v, err = getState4(ctx, a, b, c, d)
	assert.Equal(t, v, 3)
	assert.NilError(t, err)

	d <- 4
	v, err = getState4(ctx, a, b, c, d)
	assert.Equal(t, v, 4)
	assert.NilError(t, err)

	cancel()

	v, err = getState4(ctx, a, b, c, d)
	assert.Equal(t, v, 0)
	assert.Equal(t, err, context.Canceled)
}
