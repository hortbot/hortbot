package ircx

import (
	"testing"

	"github.com/jakebailey/irc"
	"gotest.tools/v3/assert"
)

func TestNormalizeChannel(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"", ""},
		{"#", ""},
		{"#foo", "#foo"},
		{"foo", "#foo"},
	}

	for _, test := range tests {
		assert.Equal(t, NormalizeChannel(test.input), test.want)
	}
}

func TestNormalizeChannels(t *testing.T) {
	tests := []struct {
		input []string
		want  []string
	}{
		{nil, nil},
		{[]string{}, nil},
		{[]string{"#foobar"}, []string{"#foobar"}},
		{[]string{"foobar"}, []string{"#foobar"}},
		{[]string{"foobar", "#something"}, []string{"#foobar", "#something"}},
	}

	for _, test := range tests {
		assert.DeepEqual(t, NormalizeChannels(test.input...), test.want)
	}
}

func TestMessages(t *testing.T) {
	tests := []struct {
		want *irc.Message
		got  *irc.Message
	}{
		{
			&irc.Message{Command: "JOIN"},
			Join(),
		},
		{
			&irc.Message{Command: "JOIN", Params: []string{"#foobar"}},
			Join("#foobar"),
		},
		{
			&irc.Message{Command: "JOIN", Params: []string{"#foobar", "#barfoo"}},
			Join("#foobar", "#barfoo"),
		},
		{
			&irc.Message{Command: "PART"},
			Part(),
		},
		{
			&irc.Message{Command: "PART", Params: []string{"#foobar"}},
			Part("#foobar"),
		},
		{
			&irc.Message{Command: "PART", Params: []string{"#foobar", "#barfoo"}},
			Part("#foobar", "#barfoo"),
		},
		{
			&irc.Message{Command: "PRIVMSG", Params: []string{"#foobar"}, Trailing: "test message"},
			PrivMsg("#foobar", "test message"),
		},
		{
			&irc.Message{Command: "PASS", Params: []string{"password"}},
			Pass("password"),
		},
		{
			&irc.Message{Command: "NICK", Params: []string{"nickname"}},
			Nick("nickname"),
		},
		{
			&irc.Message{Command: "CAP", Params: []string{"REQ"}},
			CapReq(),
		},
		{
			&irc.Message{Command: "CAP", Params: []string{"REQ"}, Trailing: "cool.test/1"},
			CapReq("cool.test/1"),
		},
		{
			&irc.Message{Command: "CAP", Params: []string{"REQ"}, Trailing: "cool.test/1 cool.test/2"},
			CapReq("cool.test/1", "cool.test/2"),
		},
		{
			&irc.Message{Command: "QUIT"},
			Quit(),
		},
	}

	for _, test := range tests {
		assert.DeepEqual(t, test.want, test.got)
	}
}
