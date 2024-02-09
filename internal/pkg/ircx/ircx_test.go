package ircx

import (
	"reflect"
	"testing"

	"github.com/hortbot/hortbot/internal/pkg/assertx"
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
			&irc.Message{Command: "JOIN", Params: []string{"#foobar"}},
			Join("#foobar"),
		},
		{
			&irc.Message{Command: "JOIN", Params: []string{"#foobar,#barfoo"}},
			Join("#foobar", "#barfoo"),
		},
		{
			&irc.Message{Command: "PART", Params: []string{"#foobar"}},
			Part("#foobar"),
		},
		{
			&irc.Message{Command: "PART", Params: []string{"#foobar,#barfoo"}},
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

func TestBadJoinPart(t *testing.T) {
	assertx.Panic(t, func() {
		Join()
	}, "must provide at least one channel")

	assertx.Panic(t, func() {
		Part()
	}, "must provide at least one channel")
}

func TestClone(t *testing.T) {
	tests := []*irc.Message{
		nil,
		{},
		{Command: "what"},
		{
			Tags: map[string]string{},
		},
		{
			Tags: map[string]string{
				"a": "b",
				"c": "d",
			},
		},
		Join("foo", "bar"),
	}

	for _, m := range tests {
		name := "nil"
		if m != nil {
			name = m.String()
		}

		t.Run(name, func(t *testing.T) {
			clone := Clone(m)

			if m == nil {
				assert.Assert(t, clone == nil)
				return
			}

			assert.Assert(t, m != clone)

			if m.Tags == nil {
				assert.Assert(t, clone.Tags == nil)
			} else {
				assert.Assert(t, reflect.ValueOf(m.Tags).Pointer() != reflect.ValueOf(clone.Tags).Pointer())
			}

			if m.Params == nil {
				assert.Assert(t, clone.Params == nil)
			} else {
				assert.Assert(t, reflect.ValueOf(m.Params).Pointer() != reflect.ValueOf(clone.Params).Pointer())
			}
		})
	}
}
