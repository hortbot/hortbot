package botstate_test

import (
	"context"
	"testing"
	"time"

	"github.com/hortbot/hortbot/internal/db/botstate"
	jsonx "github.com/hortbot/hortbot/internal/pkg/jsonx"
	"gotest.tools/v3/assert"
)

func TestAuthState(t *testing.T) {
	t.Parallel()

	db, clk := freshStore(t)

	const key = "some_state"

	type value struct {
		S string
		X int
		T time.Time
	}

	orig := &value{
		S: "string",
		X: 1234,
		T: time.Time{}.Add(time.Hour),
	}

	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()

	var got value
	ok, err := db.GetAuthState(ctx, key, &got)
	assert.NilError(t, err)
	assert.Equal(t, ok, false)

	err = db.SetAuthState(ctx, key, orig, time.Hour)
	assert.NilError(t, err)

	clk.Advance(time.Hour / 2)

	got = value{}
	ok, err = db.GetAuthState(ctx, key, &got)
	assert.NilError(t, err)
	assert.Equal(t, ok, true)
	assert.DeepEqual(t, &got, orig)

	got = value{}
	ok, err = db.GetAuthState(ctx, key, &got)
	assert.NilError(t, err)
	assert.Equal(t, ok, false, "auth state must be consumed by Get")

	err = db.SetAuthState(ctx, key, orig, time.Hour)
	assert.NilError(t, err)

	clk.Advance(time.Hour * 2)

	got = value{}
	ok, err = db.GetAuthState(ctx, key, &got)
	assert.NilError(t, err)
	assert.Equal(t, ok, false, "expired auth state must not be returned")
}

func TestAuthStateUnmarshallable(t *testing.T) {
	t.Parallel()

	db, _ := freshStore(t)
	_ = botstate.WithNow // keep import linter quiet across files

	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()

	err := db.SetAuthState(ctx, "what", jsonx.Unmarshallable(), time.Minute)
	assert.ErrorContains(t, err, jsonx.ErrUnmarshallable.Error())
}
