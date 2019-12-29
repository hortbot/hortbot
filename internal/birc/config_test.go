package birc

import (
	"regexp"
	"testing"
	"time"

	"gotest.tools/v3/assert"
)

var justinfanRegex = regexp.MustCompile(`^justinfan\d+$`)

func TestUserConfig(t *testing.T) {
	t.Run("Default", func(t *testing.T) {
		u := UserConfig{}
		u.setup()

		assert.Assert(t, justinfanRegex.MatchString(u.Nick))
		assert.Assert(t, u.Pass == "")
		assert.Assert(t, u.ReadOnly)
	})

	t.Run("With nick", func(t *testing.T) {
		u := UserConfig{
			Nick:     "FooBar",
			Pass:     "oauth:qwertyuiop1234567890",
			ReadOnly: false,
		}
		u.setup()

		assert.Assert(t, u.Nick == "foobar")
		assert.Assert(t, u.Pass == "oauth:qwertyuiop1234567890")
		assert.Assert(t, u.ReadOnly == false)
	})
}

func TestConfig(t *testing.T) {
	t.Run("Default", func(t *testing.T) {
		c := Config{}
		c.setup()

		assert.Assert(t, c.Dialer == &DefaultDialer)
		assert.Assert(t, len(c.InitialChannels) == 0)
		assert.Assert(t, len(c.Caps) == 0)
		assert.Assert(t, c.RecvBuffer == 0)
	})

	t.Run("Custom Dialer", func(t *testing.T) {
		d := &Dialer{}

		c := Config{
			Dialer: d,
		}
		c.setup()

		assert.Assert(t, c.Dialer == d)
	})

	t.Run("Negative RecvBuffer", func(t *testing.T) {
		c := Config{
			RecvBuffer: -1,
		}
		c.setup()

		assert.Assert(t, c.RecvBuffer == 0)
	})

	t.Run("Ping", func(t *testing.T) {
		test := func(interval, intervalWant time.Duration, deadline, deadlineWant time.Duration) func(t *testing.T) {
			return func(t *testing.T) {
				c := Config{
					PingInterval: interval,
					PingDeadline: deadline,
				}
				c.setup()
				assert.Equal(t, c.PingInterval, intervalWant)
				assert.Equal(t, c.PingDeadline, deadlineWant)
			}
		}

		t.Run("Negative interval", test(-1, 0, 0, 0))
		t.Run("Negative deadline", test(10, 0, -1, 0))
	})
}
