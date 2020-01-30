package bot

import (
	"context"
	"testing"

	"github.com/hortbot/hortbot/internal/pkg/assertx"
)

func TestVerifyHandlerMapEntry(t *testing.T) {
	assertx.Panic(t, func() {
		verifyHandlerMapEntry("", handlerFunc{})
	}, "empty name")

	assertx.Panic(t, func() {
		verifyHandlerMapEntry("FooBar", handlerFunc{})
	}, "name FooBar is not lowercase")

	assertx.Panic(t, func() {
		verifyHandlerMapEntry("foobar", handlerFunc{})
	}, "nil handler func")

	assertx.Panic(t, func() {
		verifyHandlerMapEntry("foobar", handlerFunc{
			fn: func(ctx context.Context, s *session, cmd, args string) error { return nil },
		})
	}, "unknown minLevel")
}
