package irctobot_test

import (
	"testing"
	"time"

	"github.com/hortbot/hortbot/internal/bot"
	"github.com/hortbot/hortbot/internal/bot/irctobot"
	"github.com/hortbot/hortbot/internal/pkg/assertx"
	"github.com/jakebailey/irc"
	"gotest.tools/v3/assert"
)

func TestIRCToMessageNil(t *testing.T) {
	msg := irctobot.ToMessage("origin", nil)
	assert.Equal(t, msg, bot.Message(nil))
}

func TestIRCToMessageNotPrivmsg(t *testing.T) {
	m := &irc.Message{Command: "NOTPRIVMSG"}
	assertx.Panic(t, func() {
		irctobot.ToMessage("origin", m)
	}, "irctobot: NOTPRIVMSG is not a PRIVMSG command")
}

func TestIRCToMessage(t *testing.T) {
	m := &irc.Message{
		Command: "PRIVMSG",
		Tags: map[string]string{
			"tag":          "value",
			"room-id":      "123",
			"user-id":      "456",
			"display-name": "display",
		},
		Prefix:   irc.Prefix{Name: "prefix"},
		Params:   []string{"#channel"},
		Trailing: "message",
	}

	msg := irctobot.ToMessage("origin", m)

	assert.Equal(t, msg.Origin(), "origin")
	assert.Equal(t, msg.BroadcasterLogin(), "channel")
	assert.Equal(t, msg.UserLogin(), "prefix")
	assert.Equal(t, msg.UserDisplay(), "display")
	assert.Equal(t, msg.BroadcasterID(), int64(123))
	assert.Equal(t, msg.UserID(), int64(456))
	assert.Equal(t, msg.Timestamp(), time.Time{})
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

	msg := irctobot.ToMessage("origin", m)

	assert.Equal(t, msg.UserDisplay(), "prefix")

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

	msg := irctobot.ToMessage("origin", m)
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

	msg := irctobot.ToMessage("origin", m)
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

	msg := irctobot.ToMessage("origin", m)
	message, me := msg.Message()
	assert.Equal(t, message, "")
	assert.Equal(t, me, false)
}

func TestNoBroadcasterID(t *testing.T) {
	m := &irc.Message{
		Command: "PRIVMSG",
		Tags:    map[string]string{"tag": "value"},
	}

	msg := irctobot.ToMessage("origin", m)
	assertx.Panic(t, func() {
		msg.BroadcasterID()
	}, "irctobot: no room-id")
}

func TestBadBroadcasterID(t *testing.T) {
	m := &irc.Message{
		Command: "PRIVMSG",
		Tags:    map[string]string{"tag": "value", "room-id": "notanint"},
	}

	msg := irctobot.ToMessage("origin", m)
	assertx.Panic(t, func() {
		msg.BroadcasterID()
	}, "irctobot: invalid room-id")
}

func TestNoUserID(t *testing.T) {
	m := &irc.Message{
		Command: "PRIVMSG",
		Tags:    map[string]string{"tag": "value"},
	}

	msg := irctobot.ToMessage("origin", m)
	assertx.Panic(t, func() {
		msg.UserID()
	}, "irctobot: no user-id")
}

func TestBadUserID(t *testing.T) {
	m := &irc.Message{
		Command: "PRIVMSG",
		Tags:    map[string]string{"tag": "value", "user-id": "notanint"},
	}

	msg := irctobot.ToMessage("origin", m)
	assertx.Panic(t, func() {
		msg.UserID()
	}, "irctobot: invalid user-id")
}

func TestEmoteCountEmpty(t *testing.T) {
	m := &irc.Message{
		Command: "PRIVMSG",
		Tags:    map[string]string{"tag": "value"},
	}

	msg := irctobot.ToMessage("origin", m)
	assert.Equal(t, msg.EmoteCount(), 0)
}

func TestEmoteCount(t *testing.T) {
	m := &irc.Message{
		Command: "PRIVMSG",
		Tags:    map[string]string{"tag": "value", "emotes": "0:1-2"},
	}

	msg := irctobot.ToMessage("origin", m)
	assert.Equal(t, msg.EmoteCount(), 1)
}

func TestTimestamp(t *testing.T) {
	m := &irc.Message{
		Command: "PRIVMSG",
		Tags:    map[string]string{"tag": "value", "tmi-sent-ts": "123456789"},
	}

	msg := irctobot.ToMessage("origin", m)
	assert.Equal(t, msg.Timestamp().Unix(), int64(123456))
}

func TestTimestampEmpty(t *testing.T) {
	m := &irc.Message{
		Command: "PRIVMSG",
		Tags:    map[string]string{"tag": "value"},
	}

	msg := irctobot.ToMessage("origin", m)
	assert.Equal(t, msg.Timestamp(), time.Time{})
}

func TestIDEmpty(t *testing.T) {
	m := &irc.Message{
		Command: "PRIVMSG",
		Tags:    map[string]string{"tag": "value"},
	}

	msg := irctobot.ToMessage("origin", m)
	assert.Equal(t, msg.ID(), "")
}

func TestID(t *testing.T) {
	m := &irc.Message{
		Command: "PRIVMSG",
		Tags:    map[string]string{"tag": "value", "id": "123"},
	}

	msg := irctobot.ToMessage("origin", m)
	assert.Equal(t, msg.ID(), "123")
}
