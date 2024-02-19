package must_test

import (
	"errors"
	"testing"

	"github.com/hortbot/hortbot/internal/pkg/assertx"
	"github.com/hortbot/hortbot/internal/pkg/must"
	"gotest.tools/v3/assert"
)

func TestMustOK(t *testing.T) {
	t.Parallel()

	want := 1234
	got := must.Must(want, nil)

	assert.Equal(t, want, got)
}

func TestMustPanic(t *testing.T) {
	t.Parallel()

	err := errors.New("an error")

	assertx.Panic(t, func() {
		must.Must(1234, err)
	}, err)
}

func TestNilErrorOK(t *testing.T) {
	t.Parallel()

	must.NilError(nil)
}

func TestNilErrorPanic(t *testing.T) {
	t.Parallel()

	err := errors.New("an error")

	assertx.Panic(t, func() {
		must.NilError(err)
	}, err)
}
