package fakeirc

import (
	"testing"
	"time"

	"gotest.tools/assert"
	"gotest.tools/env"
)

func TestHelperSleepDur(t *testing.T) {
	dur, err := getSleepDur()
	assert.NilError(t, err)
	assert.Equal(t, dur, DefaultSleepDur)
}

func TestHelperSleepDurEnv(t *testing.T) {
	defer env.Patch(t, sleepEnvVarName, "100ms")()

	dur, err := getSleepDur()
	assert.NilError(t, err)
	assert.Equal(t, dur, 100*time.Millisecond)
}

func TestHelperSleepDurEnvBad(t *testing.T) {
	defer env.Patch(t, sleepEnvVarName, "100")()

	_, err := getSleepDur()
	assert.Assert(t, err != nil)
}
