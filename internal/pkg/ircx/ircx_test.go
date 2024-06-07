package ircx

import (
	"testing"

	"github.com/jakebailey/irc"
	"gotest.tools/v3/assert"
)

func TestMessages(t *testing.T) {
	t.Parallel()
	tests := []struct {
		want *irc.Message
		got  *irc.Message
	}{
		{
			&irc.Message{Command: "PRIVMSG", Params: []string{"#foobar"}, Trailing: "test message"},
			PrivMsg("#foobar", "test message"),
		},
	}

	for _, test := range tests {
		assert.DeepEqual(t, test.want, test.got)
	}
}
