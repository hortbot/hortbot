package correlation_test

import (
	"testing"

	"github.com/docker/distribution/context"
	"github.com/gotestyourself/gotestyourself/assert"
	"github.com/hortbot/hortbot/internal/pkg/correlation"
	"github.com/rs/xid"
)

func TestFromWith(t *testing.T) {
	ctx := context.Background()

	id := correlation.FromContext(ctx)
	assert.Assert(t, id.IsNil())

	ctx = correlation.With(ctx)
	id1 := correlation.FromContext(ctx)

	assert.Assert(t, !id1.IsNil())

	ctx = correlation.With(ctx)
	id2 := correlation.FromContext(ctx)

	assert.Equal(t, id1, id2)
}

func TestFromWithID(t *testing.T) {
	ctx := context.Background()

	id1 := xid.New()
	ctx = correlation.WithID(ctx, id1)

	id := correlation.FromContext(ctx)
	assert.Equal(t, id, id1)

	id2 := xid.New()
	ctx = correlation.WithID(ctx, id2)

	id = correlation.FromContext(ctx)
	assert.Equal(t, id, id2)

	ctx = correlation.With(ctx)

	id = correlation.FromContext(ctx)
	assert.Equal(t, id, id2)
}
