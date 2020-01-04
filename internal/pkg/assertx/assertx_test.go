package assertx_test

import (
	"testing"

	"github.com/hortbot/hortbot/internal/pkg/assertx"
	"github.com/hortbot/hortbot/internal/pkg/assertx/assertxfakes"
	"gotest.tools/v3/assert"
)

func TestPanic(t *testing.T) {
	t.Run("No panic", func(t *testing.T) {
		fake := &assertxfakes.FakeTestingT{}

		assertx.Panic(fake, func() {}, nil)

		assert.Assert(t, fake.HelperCallCount() > 0)
		assert.Equal(t, fake.FailCallCount(), 0)
		assert.Equal(t, fake.FailNowCallCount(), 0)
	})

	t.Run("Panic pass", func(t *testing.T) {
		fake := &assertxfakes.FakeTestingT{}
		v := "some panic value"

		assertx.Panic(fake, func() {
			panic(v)
		}, v)

		assert.Assert(t, fake.HelperCallCount() > 0)
		assert.Equal(t, fake.FailCallCount(), 0)
		assert.Equal(t, fake.FailNowCallCount(), 0)
	})

	t.Run("Panic fail", func(t *testing.T) {
		fake := &assertxfakes.FakeTestingT{}
		v := "some panic value"

		assertx.Panic(fake, func() {
			panic("sorry")
		}, v)

		assert.Assert(t, fake.HelperCallCount() > 0)
		assert.Assert(t, fake.FailNowCallCount() > 0)
	})
}
