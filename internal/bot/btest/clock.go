package btest

import (
	"context"
	"testing"
	"time"

	"github.com/leononame/clock"
	"gotest.tools/assert"
)

func (st *scriptTester) clockForward(t testing.TB, _, args string, lineNum int) {
	if _, ok := st.bc.Clock.(*clock.Mock); !ok {
		t.Fatalf("clock must be a mock: line %d", lineNum)
	}

	dur, err := time.ParseDuration(args)
	assert.NilError(t, err, "line %d", lineNum)

	st.addAction(func(ctx context.Context) {
		st.clock.Forward(dur)
		st.redis.FastForward(dur)
	})
}

func (st *scriptTester) clockSet(t testing.TB, _, args string, lineNum int) {
	if _, ok := st.bc.Clock.(*clock.Mock); !ok {
		t.Fatalf("clock must be a mock: line %d", lineNum)
	}

	var tm time.Time

	if args == "now" {
		tm = time.Now()
	} else {
		var err error
		tm, err = time.Parse(time.RFC3339, args)
		assert.NilError(t, err, "line %d", lineNum)
	}

	st.addAction(func(ctx context.Context) {
		st.clock.Set(tm)
		st.redis.SetTime(tm)
	})
}

func (st *scriptTester) sleep(t testing.TB, _, args string, lineNum int) {
	dur, err := time.ParseDuration(args)
	assert.NilError(t, err, "line %d", lineNum)

	st.addAction(func(ctx context.Context) {
		time.Sleep(dur)
	})
}
