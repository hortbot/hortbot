package bnsqmeta_test

import (
	"context"
	"testing"
	"time"

	"github.com/hortbot/hortbot/internal/bnsq/bnsqmeta"
	"gotest.tools/v3/assert"
)

func TestTimestamp(t *testing.T) {
	ctx := context.Background()

	ts := bnsqmeta.Timestamp(ctx)
	assert.Assert(t, ts.IsZero())

	ts1 := time.Now()
	ctx = bnsqmeta.WithTimestamp(ctx, ts1)
	ts = bnsqmeta.Timestamp(ctx)
	assert.Assert(t, ts.Equal(ts1))

	ts2 := ts1.Add(time.Hour)
	ctx = bnsqmeta.WithTimestamp(ctx, ts2)
	ts = bnsqmeta.Timestamp(ctx)
	assert.Assert(t, ts.Equal(ts2))
}
