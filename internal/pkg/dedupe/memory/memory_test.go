package memory_test

import (
	"testing"
	"time"

	"github.com/hortbot/hortbot/internal/pkg/dedupe/memory"
	"gotest.tools/v3/assert"
)

const id = "id"

func TestCheckNotFound(t *testing.T) {
	d := memory.New(time.Second, time.Second)
	defer d.Stop()

	seen, err := d.Check(id)
	assert.Assert(t, !seen)
	assert.NilError(t, err)
}

func TestMarkThenCheck(t *testing.T) {
	d := memory.New(time.Second, time.Second)
	defer d.Stop()

	assert.NilError(t, d.Mark(id))

	seen, err := d.Check(id)
	assert.Assert(t, seen)
	assert.NilError(t, err)
}

func TestCheckAndMark(t *testing.T) {
	d := memory.New(time.Second, time.Second)
	defer d.Stop()

	seen, err := d.CheckAndMark(id)
	assert.Assert(t, !seen)
	assert.NilError(t, err)

	seen, err = d.Check(id)
	assert.Assert(t, seen)
	assert.NilError(t, err)
}

func TestCheckAndMarkTwice(t *testing.T) {
	d := memory.New(time.Second, time.Second)
	defer d.Stop()

	seen, err := d.CheckAndMark(id)
	assert.Assert(t, !seen)
	assert.NilError(t, err)

	seen, err = d.CheckAndMark(id)
	assert.Assert(t, seen)
	assert.NilError(t, err)
}

func TestExpire(t *testing.T) {
	d := memory.New(time.Millisecond, 10*time.Millisecond)
	defer d.Stop()

	seen, err := d.CheckAndMark(id)
	assert.Assert(t, !seen)
	assert.NilError(t, err)

	time.Sleep(20 * time.Millisecond)

	seen, err = d.Check(id)
	assert.Assert(t, !seen)
	assert.NilError(t, err)
}
