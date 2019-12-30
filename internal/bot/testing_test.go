package bot

import (
	"testing"

	"github.com/hortbot/hortbot/internal/pkg/assertx"
)

func TestTestingHelper(t *testing.T) {
	t.Run("InTest", func(t *testing.T) {
		helper := &testingHelper{}

		// Normally fails, but if nil should be ignored.
		helper.checkUserNameID("foo", 1)
		helper.checkUserNameID("foo", 1)

		assertx.Panic(t, func() {
			helper.checkUserNameID("foo", 2)
		}, testingPanic("foo previously had id 1, now 2"))

		assertx.Panic(t, func() {
			helper.checkUserNameID("bar", 1)
		}, testingPanic("1 previously had name foo, now bar"))
	})

	t.Run("Nil", func(t *testing.T) {
		var helper *testingHelper

		helper.checkUserNameID("foo", 1)
		helper.checkUserNameID("foo", 1)
		helper.checkUserNameID("foo", 2)
		helper.checkUserNameID("bar", 1)
	})
}
