package irctobot_test

import (
	"testing"

	"github.com/hortbot/hortbot/internal/bot"
	"github.com/hortbot/hortbot/internal/bot/irctobot"
	"github.com/hortbot/hortbot/internal/pkg/assertx"
	"github.com/jakebailey/irc"
	"gotest.tools/v3/assert"
)

func TestIRCToMessageNil(t *testing.T) {
	msg := irctobot.IRCToMessage(nil)
	assert.Equal(t, msg, bot.Message(nil))
}

func TestIRCToMessageNotPrivmsg(t *testing.T) {
	m := &irc.Message{Command: "NOTPRIVMSG"}
	assertx.Panic(t, func() {
		irctobot.IRCToMessage(m)
	}, "irctobot: NOTPRIVMSG is not a PRIVMSG command")
}

func TestIRCToMessage(t *testing.T) {
	m := &irc.Message{
		Command:  "PRIVMSG",
		Tags:     map[string]string{"tag": "value"},
		Prefix:   irc.Prefix{Name: "prefix"},
		Params:   []string{"#channel"},
		Trailing: "message",
	}

	msg := irctobot.IRCToMessage(m)

	assert.DeepEqual(t, msg.Tags(), map[string]string{"tag": "value"})
	assert.Equal(t, msg.BroadcasterLogin(), "channel")
	assert.Equal(t, msg.UserLogin(), "prefix")
	message, me := msg.Message()
	assert.Equal(t, message, "message")
	assert.Equal(t, me, false)
}

func TestIRCToMessageAction(t *testing.T) {
	m := &irc.Message{
		Command:  "PRIVMSG",
		Tags:     map[string]string{"tag": "value"},
		Prefix:   irc.Prefix{Name: "prefix"},
		Params:   []string{"#channel"},
		Trailing: "\x01ACTION message\x01",
	}

	msg := irctobot.IRCToMessage(m)

	assert.DeepEqual(t, msg.Tags(), map[string]string{"tag": "value"})
	assert.Equal(t, msg.BroadcasterLogin(), "channel")
	assert.Equal(t, msg.UserLogin(), "prefix")
	message, me := msg.Message()
	assert.Equal(t, message, "message")
	assert.Equal(t, me, true)
}

func TestBroadcasterLoginNoParams(t *testing.T) {
	m := &irc.Message{
		Command:  "PRIVMSG",
		Tags:     map[string]string{"tag": "value"},
		Prefix:   irc.Prefix{Name: "prefix"},
		Trailing: "\x01ACTION message\x01",
	}

	msg := irctobot.IRCToMessage(m)
	assertx.Panic(t, func() {
		msg.BroadcasterLogin()
	}, "irctobot: no params")
}

func TestBroadcasterLoginNoChannel(t *testing.T) {
	m := &irc.Message{
		Command:  "PRIVMSG",
		Tags:     map[string]string{"tag": "value"},
		Prefix:   irc.Prefix{Name: "prefix"},
		Params:   []string{"nochannel"},
		Trailing: "\x01ACTION message\x01",
	}

	msg := irctobot.IRCToMessage(m)
	assertx.Panic(t, func() {
		msg.BroadcasterLogin()
	}, "irctobot: no channel")
}

func TestMessageNonActionCTCP(t *testing.T) {
	m := &irc.Message{
		Command:  "PRIVMSG",
		Tags:     map[string]string{"tag": "value"},
		Prefix:   irc.Prefix{Name: "prefix"},
		Params:   []string{"#channel"},
		Trailing: "\x01NOTACTION message\x01",
	}

	msg := irctobot.IRCToMessage(m)
	message, me := msg.Message()
	assert.Equal(t, message, "")
	assert.Equal(t, me, false)
}
