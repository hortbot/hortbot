package redis_test

import (
	"context"
	"testing"
	"time"

	"github.com/hortbot/hortbot/internal/db/redis"
	"github.com/hortbot/hortbot/internal/pkg/testutil/miniredistest"
	"github.com/leononame/clock"
	"gotest.tools/v3/assert"
)

func TestBot(t *testing.T) {
	t.Parallel()

	_, c, cleanup, err := miniredistest.New()
	assert.NilError(t, err)
	defer cleanup()

	db := redis.New(c)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// All of these are tested in other packages.
	// Just verify that they don't crash.

	err = db.LinkPermit(ctx, "foo", "bar", time.Minute)
	assert.NilError(t, err)

	_, err = db.HasLinkPermit(ctx, "foo", "bar")
	assert.NilError(t, err)

	_, err = db.Confirm(ctx, "foo", "bar", "baz", time.Minute)
	assert.NilError(t, err)

	_, err = db.RepeatAllowed(ctx, "foo", 123, time.Minute)
	assert.NilError(t, err)

	_, err = db.ScheduledAllowed(ctx, "foo", 123, time.Minute)
	assert.NilError(t, err)

	_, err = db.AutoreplyAllowed(ctx, "foo", 123, time.Minute)
	assert.NilError(t, err)

	_, err = db.FilterWarned(ctx, "foo", "user", "filter", time.Minute)
	assert.NilError(t, err)

	err = db.RaffleAdd(ctx, "foo", "user")
	assert.NilError(t, err)

	err = db.RaffleReset(ctx, "foo")
	assert.NilError(t, err)

	_, _, err = db.RaffleWinner(ctx, "foo")
	assert.NilError(t, err)

	_, err = db.RaffleCount(ctx, "foo")
	assert.NilError(t, err)

	err = db.MarkCooldown(ctx, "foo", "bar", time.Minute)
	assert.NilError(t, err)

	_, err = db.CheckAndMarkCooldown(ctx, "foo", "bar", time.Minute)
	assert.NilError(t, err)
}

func TestLinkPermit(t *testing.T) {
	t.Parallel()

	s, c, cleanup, err := miniredistest.New()
	assert.NilError(t, err)
	defer cleanup()

	db := redis.New(c)
	clk := clock.NewMock()
	s.SetTime(clk.Now())

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	const (
		channel = "1234"
		user    = "user1"
	)

	allowed, err := db.HasLinkPermit(ctx, channel, user)
	assert.NilError(t, err)
	assert.Equal(t, allowed, false)

	err = db.LinkPermit(ctx, channel, user, 10*time.Second)
	assert.NilError(t, err)

	forward(s, clk, time.Hour)

	allowed, err = db.HasLinkPermit(ctx, channel, user)
	assert.NilError(t, err)
	assert.Equal(t, allowed, false)

	err = db.LinkPermit(ctx, channel, user, 10*time.Second)
	assert.NilError(t, err)

	forward(s, clk, time.Second)

	allowed, err = db.HasLinkPermit(ctx, channel, user)
	assert.NilError(t, err)
	assert.Equal(t, allowed, true)

	allowed, err = db.HasLinkPermit(ctx, channel, "nobody")
	assert.NilError(t, err)
	assert.Equal(t, allowed, false)

	allowed, err = db.HasLinkPermit(ctx, "nobody", user)
	assert.NilError(t, err)
	assert.Equal(t, allowed, false)

	forward(s, clk, time.Second)

	allowed, err = db.HasLinkPermit(ctx, channel, user)
	assert.NilError(t, err)
	assert.Equal(t, allowed, false)
}

func TestUserState(t *testing.T) {
	t.Parallel()

	s, c, cleanup, err := miniredistest.New()
	assert.NilError(t, err)
	defer cleanup()

	db := redis.New(c)
	clk := clock.NewMock()
	s.SetTime(clk.Now())

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	const (
		botName = "hortbot"
		channel = "#foobar"
	)

	fast, err := db.GetUserState(ctx, botName, channel)
	assert.NilError(t, err)
	assert.Equal(t, fast, false)

	err = db.SetUserState(ctx, botName, channel, true, time.Minute)
	assert.NilError(t, err)

	forward(s, clk, time.Second)

	fast, err = db.GetUserState(ctx, botName, channel)
	assert.NilError(t, err)
	assert.Equal(t, fast, true)

	forward(s, clk, time.Hour)

	err = db.SetUserState(ctx, botName, channel, true, time.Minute)
	assert.NilError(t, err)

	forward(s, clk, time.Second)

	fast, err = db.GetUserState(ctx, botName, channel)
	assert.NilError(t, err)
	assert.Equal(t, fast, true)

	err = db.SetUserState(ctx, botName, channel, false, time.Minute)
	assert.NilError(t, err)

	forward(s, clk, time.Second)

	fast, err = db.GetUserState(ctx, botName, channel)
	assert.NilError(t, err)
	assert.Equal(t, fast, false)
}
