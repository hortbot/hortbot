package fakeirc

import (
	"testing"
	"time"

	"gotest.tools/v3/assert"
)

func TestHelperSleepDur(t *testing.T) {
	dur, err := getSleepDur()
	assert.NilError(t, err)
	assert.Equal(t, dur, defaultSleepDur)
}

func TestHelperSleepDurEnv(t *testing.T) {
	t.Setenv(sleepEnvVarName, "100ms")

	dur, err := getSleepDur()
	assert.NilError(t, err)
	assert.Equal(t, dur, 100*time.Millisecond)
}

func TestHelperSleepDurEnvBad(t *testing.T) {
	t.Setenv(sleepEnvVarName, "100")

	_, err := getSleepDur()
	assert.Assert(t, err != nil)
}
