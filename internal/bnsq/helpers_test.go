package bnsq_test

import (
	"context"
	"testing"
	"time"

	"github.com/hortbot/hortbot/internal/pkg/testutil"
	"github.com/zikaeroh/ctxlog"
)

func testContext(t testing.TB) (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)

	logger := testutil.Logger(t)
	ctx = ctxlog.WithLogger(ctx, logger)

	return ctx, cancel
}
