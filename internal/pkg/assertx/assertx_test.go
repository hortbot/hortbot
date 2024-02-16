package assertx_test

import (
	"testing"

	"github.com/hortbot/hortbot/internal/pkg/assertx"
	"github.com/hortbot/hortbot/internal/pkg/assertx/assertxmocks"
	"gotest.tools/v3/assert"
)

func TestPanic(t *testing.T) {
	t.Run("No panic", func(t *testing.T) {
		fake := &assertxmocks.TestingTMock{
			HelperFunc:  func() {},
			FailFunc:    func() {},
			FailNowFunc: func() {},
		}

		assertx.Panic(fake, func() {}, nil)

		assert.Assert(t, len(fake.HelperCalls()) > 0)
		assert.Equal(t, len(fake.FailCalls()), 0)
		assert.Equal(t, len(fake.FailNowCalls()), 0)
	})

	t.Run("Panic pass", func(t *testing.T) {
		fake := &assertxmocks.TestingTMock{
			HelperFunc:  func() {},
			FailFunc:    func() {},
			FailNowFunc: func() {},
		}
		v := "some panic value"

		assertx.Panic(fake, func() {
			panic(v)
		}, v)

		assert.Assert(t, len(fake.HelperCalls()) > 0)
		assert.Equal(t, len(fake.FailCalls()), 0)
		assert.Equal(t, len(fake.FailNowCalls()), 0)
	})

	t.Run("Panic fail", func(t *testing.T) {
		fake := &assertxmocks.TestingTMock{
			HelperFunc:  func() {},
			LogFunc:     func(args ...interface{}) {},
			FailNowFunc: func() {},
		}
		v := "some panic value"

		assertx.Panic(fake, func() {
			panic("sorry")
		}, v)

		assert.Assert(t, len(fake.HelperCalls()) > 0)
		assert.Assert(t, len(fake.FailNowCalls()) > 0)
	})
}
