package btest

import (
	"context"
	"testing"
	"testing/synctest"
	"time"

	"gotest.tools/v3/assert"
)

func (st *scriptTester) clockForward(t testing.TB, _, args string, lineNum int) {
	dur, err := time.ParseDuration(args)
	assert.NilError(t, err, "line %d", lineNum)

	st.addAction(func(ctx context.Context) {
		time.Sleep(dur)
		st.redisServer.FastForward(dur)
		st.redisServer.SetTime(time.Now())
		synctest.Wait()
	})
}

func (st *scriptTester) clockSet(t testing.TB, _, args string, lineNum int) {
	var tm time.Time

	if args == "now" {
		// Inside the synctest bubble, time.Now() returns fake time.
		// "now" means the real wall-clock time; we captured it before
		// entering the bubble and stored it in st.realNow.
		tm = st.realNow
	} else {
		var err error
		tm, err = time.Parse(time.RFC3339, args)
		assert.NilError(t, err, "line %d", lineNum)
	}

	st.addAction(func(ctx context.Context) {
		diff := time.Until(tm)
		if diff < 0 {
			t.Fatalf("line %d: clock_set cannot move time backward (target %v is %v before now)", lineNum, tm, -diff)
		}
		if diff > 0 {
			time.Sleep(diff)
		}
		st.redisServer.SetTime(time.Now())
		synctest.Wait()
	})
}

func (st *scriptTester) sleep(t testing.TB, _, args string, lineNum int) {
	dur, err := time.ParseDuration(args)
	assert.NilError(t, err, "line %d", lineNum)

	st.addAction(func(ctx context.Context) {
		time.Sleep(dur)
	})
}
