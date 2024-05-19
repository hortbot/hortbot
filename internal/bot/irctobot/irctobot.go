package irctobot

import (
	"strings"

	"github.com/hortbot/hortbot/internal/bot"
	"github.com/jakebailey/irc"
)

type ircMessage struct {
	m *irc.Message
}

func IRCToMessage(m *irc.Message) bot.Message {
	if m == nil {
		return nil
	}
	if m.Command != "PRIVMSG" {
		panic("irctobot: " + m.Command + " is not a PRIVMSG command")
	}
	return &ircMessage{m: m}
}

func (m *ircMessage) Tags() map[string]string { return m.m.Tags }

func (m *ircMessage) BroadcasterLogin() string {
	if len(m.m.Params) == 0 {
		panic("irctobot: no params")
	}

	channel := m.m.Params[0]
	name, ok := strings.CutPrefix(channel, "#")
	if !ok {
		panic("irctobot: no channel")
	}

	return strings.ToLower(name)
}

func (m *ircMessage) UserLogin() string { return strings.ToLower(m.m.Prefix.Name) }

func (m *ircMessage) Message() (message string, me bool) {
	message = m.m.Trailing

	if c, a, ok := irc.ParseCTCP(message); ok {
		if c != "ACTION" {
			return "", false
		}

		message = a
		me = true
	}

	return strings.TrimSpace(message), me
}
