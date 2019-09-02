package rdb_test

import (
	"testing"
	"time"

	"github.com/hortbot/hortbot/internal/pkg/rdb"
	"github.com/hortbot/hortbot/internal/pkg/testutil/miniredistest"
	"github.com/leononame/clock"
	"gotest.tools/v3/assert"
)

func TestRateLimit(t *testing.T) {
	// t.Parallel()

	type pair struct {
		dur     time.Duration
		allowed bool
	}

	tests := []struct {
		name   string
		limit  int
		window int
		checks []pair
	}{
		{
			name: "Zero limit",
			checks: []pair{
				{0, false},
			},
		},
		{
			name:  "Negative limit",
			limit: -10,
			checks: []pair{
				{0, false},
			},
		},
		{
			name:  "Zero window",
			limit: 10,
			checks: []pair{
				{0, false},
			},
		},
		{
			name:   "Negative window",
			limit:  10,
			window: -10,
			checks: []pair{
				{0, false},
			},
		},
		{
			name:   "Allowed then disallowed",
			limit:  5,
			window: 10,
			checks: []pair{
				{time.Second, true},
				{time.Second, true},
				{time.Second, true},
				{time.Second, true},
				{time.Second, true},
				{time.Second, false},
				{time.Second, false},
				{time.Second, false},
				{time.Second, false},
				{time.Second, false},
				{time.Second, true},
				{time.Second, true},
				{time.Second, true},
				{time.Second, true},
				{time.Second, true},
				{time.Second, false},
				{time.Second, false},
				{time.Second, false},
				{time.Second, false},
				{time.Second, false},
			},
		},
		{
			name:   "Sub-second",
			limit:  5,
			window: 1,
			checks: []pair{
				{time.Second / 10, true},
				{time.Second / 10, true},
				{time.Second / 10, true},
				{time.Second / 10, true},
				{time.Second / 10, true},
				{time.Second / 10, false},
				{time.Second / 10, false},
				{time.Second / 10, false},
				{time.Second / 10, false},
				{time.Second / 10, false},
				{time.Second / 10, true},
				{time.Second / 10, true},
				{time.Second / 10, true},
				{time.Second / 10, true},
				{time.Second / 10, true},
				{time.Second / 10, false},
				{time.Second / 10, false},
				{time.Second / 10, false},
				{time.Second / 10, false},
				{time.Second / 10, false},
			},
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			// t.Parallel()

			clk := clock.NewMock()

			s, c, cleanup, err := miniredistest.New()
			assert.NilError(t, err)
			defer cleanup()

			db, err := rdb.New(c)
			assert.NilError(t, err)

			for i, check := range test.checks {
				s.FastForward(check.dur)
				clk.Forward(check.dur)
				s.SetTime(clk.Now()) // Hack, as FastForward doesn't actually change TIME.

				allowed, err := db.RateLimit(test.limit, test.window, "some", "key")

				assert.NilError(t, err)
				assert.Equal(t, check.allowed, allowed, "check %d", i)
			}
		})
	}
}

func TestBadRateLimitScript(t *testing.T) {
	defer rdb.ReplaceRateLimit("local")()

	_, c, cleanup, err := miniredistest.New()
	assert.NilError(t, err)
	defer cleanup()

	_, err = rdb.New(c)
	assert.ErrorContains(t, err, "syntax error")
}
