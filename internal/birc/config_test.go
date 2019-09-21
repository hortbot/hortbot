package birc_test

import (
	"regexp"
	"testing"
	"time"

	"github.com/hortbot/hortbot/internal/birc"
	"gotest.tools/v3/assert"
)

var justinfanRegex = regexp.MustCompile(`^justinfan\d+$`)

func TestUserConfig(t *testing.T) {
	t.Run("Default", func(t *testing.T) {
		u := birc.UserConfig{}
		u.Setup()

		assert.Assert(t, justinfanRegex.MatchString(u.Nick))
		assert.Assert(t, u.Pass == "")
		assert.Assert(t, u.ReadOnly)
	})

	t.Run("With nick", func(t *testing.T) {
		u := birc.UserConfig{
			Nick:     "FooBar",
			Pass:     "oauth:qwertyuiop1234567890",
			ReadOnly: false,
		}
		u.Setup()

		assert.Assert(t, u.Nick == "foobar")
		assert.Assert(t, u.Pass == "oauth:qwertyuiop1234567890")
		assert.Assert(t, u.ReadOnly == false)
	})
}

func TestConfig(t *testing.T) {
	t.Run("Default", func(t *testing.T) {
		c := birc.Config{}
		c.Setup()

		assert.Assert(t, c.Dialer == &birc.DefaultDialer)
		assert.Assert(t, len(c.InitialChannels) == 0)
		assert.Assert(t, len(c.Caps) == 0)
		assert.Assert(t, c.RecvBuffer == 0)
	})

	t.Run("Custom Dialer", func(t *testing.T) {
		d := &birc.Dialer{}

		c := birc.Config{
			Dialer: d,
		}
		c.Setup()

		assert.Assert(t, c.Dialer == d)
	})

	t.Run("Negative RecvBuffer", func(t *testing.T) {
		c := birc.Config{
			RecvBuffer: -1,
		}
		c.Setup()

		assert.Assert(t, c.RecvBuffer == 0)
	})

	t.Run("Ping", func(t *testing.T) {
		test := func(interval, intervalWant time.Duration, deadline, deadlineWant time.Duration) func(t *testing.T) {
			return func(t *testing.T) {
				c := birc.Config{
					PingInterval: interval,
					PingDeadline: deadline,
				}
				c.Setup()
				assert.Equal(t, c.PingInterval, intervalWant)
				assert.Equal(t, c.PingDeadline, deadlineWant)
			}
		}

		t.Run("Negative interval", test(-1, 0, 0, 0))
		t.Run("Negative deadline", test(10, 0, -1, 0))
	})
}
