package redis

import (
	"context"
	"testing"
	"time"

	"github.com/hortbot/hortbot/internal/pkg/testutil/miniredistest"
	"github.com/leononame/clock"
	"gotest.tools/v3/assert"
)

func TestRateLimit(t *testing.T) {
	t.Parallel()

	type pair struct {
		dur     time.Duration
		allowed bool
	}

	tests := []struct {
		name      string
		slowLimit int
		window    time.Duration
		checks    []pair
	}{
		{
			name: "Zero limit",
			checks: []pair{
				{0, false},
			},
		},
		{
			name:      "Negative limit",
			slowLimit: -10,
			checks: []pair{
				{0, false},
			},
		},
		{
			name:      "Zero window",
			slowLimit: 10,
			checks: []pair{
				{0, false},
			},
		},
		{
			name:      "Negative window",
			slowLimit: 10,
			window:    -10 * time.Second,
			checks: []pair{
				{0, false},
			},
		},
		{
			name:      "Allowed then disallowed",
			slowLimit: 5,
			window:    10 * time.Second,
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
			name:      "Sub-second",
			slowLimit: 5,
			window:    time.Second,
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
			t.Parallel()
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			clk := clock.NewMock()

			s, c, cleanup, err := miniredistest.New()
			assert.NilError(t, err)
			defer cleanup()

			const onlyFastKey = "only_fast"

			for i, check := range test.checks {
				s.FastForward(check.dur)
				clk.Forward(check.dur)
				s.SetTime(clk.Now()) // Hack, as FastForward doesn't actually change TIME.

				allowed, err := rateLimit(ctx, c, "some:key", test.window, test.slowLimit, test.slowLimit, onlyFastKey)

				assert.NilError(t, err)
				assert.Equal(t, check.allowed, allowed, "check %d", i)
			}
		})
	}
}
