package botstate_test

import (
	"context"
	"testing"
	"time"

	"gotest.tools/v3/assert"
)

func TestCheckAndMarkCooldownOnStaleRow(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()

	db, clk := freshStore(t)

	seen, err := db.CheckAndMarkCooldown(ctx, "channel", "key", time.Second)
	assert.NilError(t, err)
	assert.Assert(t, !seen)

	clk.Advance(2 * time.Second)

	seen, err = db.CheckAndMarkCooldown(ctx, "channel", "key", 10*time.Second)
	assert.NilError(t, err)
	assert.Assert(t, !seen, "stale cooldown should be treated as absent")
}

func TestConfirmOnStaleRow(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()

	db, clk := freshStore(t)

	confirmed, err := db.Confirm(ctx, "channel", "user", "key", time.Second)
	assert.NilError(t, err)
	assert.Assert(t, !confirmed)

	clk.Advance(2 * time.Second)

	confirmed, err = db.Confirm(ctx, "channel", "user", "key", 10*time.Second)
	assert.NilError(t, err)
	assert.Assert(t, !confirmed, "stale confirm row should be treated as absent")
}

func TestConfirmLifecycle(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()

	db, clk := freshStore(t)

	confirmed, err := db.Confirm(ctx, "channel", "user", "key", 10*time.Second)
	assert.NilError(t, err)
	assert.Assert(t, !confirmed, "first call: nothing to confirm yet")

	clk.Advance(time.Second)

	confirmed, err = db.Confirm(ctx, "channel", "user", "key", 10*time.Second)
	assert.NilError(t, err)
	assert.Assert(t, confirmed, "second call within window: confirms")

	confirmed, err = db.Confirm(ctx, "channel", "user", "key", 10*time.Second)
	assert.NilError(t, err)
	assert.Assert(t, !confirmed, "third call: gate was consumed; restarts")
}

func TestCheckAndMarkCooldownLifecycle(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()

	db, clk := freshStore(t)

	seen, err := db.CheckAndMarkCooldown(ctx, "ch", "key", 10*time.Second)
	assert.NilError(t, err)
	assert.Assert(t, !seen)

	clk.Advance(5 * time.Second)

	seen, err = db.CheckAndMarkCooldown(ctx, "ch", "key", 10*time.Second)
	assert.NilError(t, err)
	assert.Assert(t, seen)

	clk.Advance(11 * time.Second)

	seen, err = db.CheckAndMarkCooldown(ctx, "ch", "key", 10*time.Second)
	assert.NilError(t, err)
	assert.Assert(t, !seen)
}

func TestMarkCooldownOverwrites(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()

	db, clk := freshStore(t)

	assert.NilError(t, db.MarkCooldown(ctx, "ch", "key", time.Second))

	clk.Advance(2 * time.Second)
	assert.NilError(t, db.MarkCooldown(ctx, "ch", "key", 10*time.Second))

	seen, err := db.CheckAndMarkCooldown(ctx, "ch", "key", time.Second)
	assert.NilError(t, err)
	assert.Assert(t, seen, "MarkCooldown should have written a live row")
}

func TestFilterWarnedRefreshesExpiry(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()

	db, clk := freshStore(t)

	warned, err := db.FilterWarned(ctx, "ch", "user", "links", time.Hour)
	assert.NilError(t, err)
	assert.Assert(t, !warned, "first call: not previously warned")

	clk.Advance(500 * time.Millisecond)

	warned, err = db.FilterWarned(ctx, "ch", "user", "links", time.Hour)
	assert.NilError(t, err)
	assert.Assert(t, warned)

	clk.Advance(2 * time.Second)

	warned, err = db.FilterWarned(ctx, "ch", "user", "links", time.Hour)
	assert.NilError(t, err)
	assert.Assert(t, warned, "refresh should extend the warning expiry")
}

func TestFilterWarnedInitialExpiryIsOneSecond(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()

	db, clk := freshStore(t)

	warned, err := db.FilterWarned(ctx, "ch", "user", "links", time.Hour)
	assert.NilError(t, err)
	assert.Assert(t, !warned)

	clk.Advance(2 * time.Second)

	warned, err = db.FilterWarned(ctx, "ch", "user", "links", time.Hour)
	assert.NilError(t, err)
	assert.Assert(t, !warned)
}

func TestCooldownKindsAreIndependent(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()

	db, _ := freshStore(t)

	allowed, err := db.RepeatAllowed(ctx, "ch", 1, time.Hour)
	assert.NilError(t, err)
	assert.Assert(t, allowed)

	allowed, err = db.ScheduledAllowed(ctx, "ch", 1, time.Hour)
	assert.NilError(t, err)
	assert.Assert(t, allowed)

	allowed, err = db.AutoreplyAllowed(ctx, "ch", 1, time.Hour)
	assert.NilError(t, err)
	assert.Assert(t, allowed)

	allowed, err = db.RepeatAllowed(ctx, "ch", 1, time.Hour)
	assert.NilError(t, err)
	assert.Assert(t, !allowed)
}
