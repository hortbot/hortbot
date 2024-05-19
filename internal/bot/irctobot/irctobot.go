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
	return &ircMessage{m: m}
}

func (m *ircMessage) Command() string         { return m.m.Command }
func (m *ircMessage) Tags() map[string]string { return m.m.Tags }
func (m *ircMessage) Params() []string        { return m.m.Params }
func (m *ircMessage) PrefixName() string      { return m.m.Prefix.Name }

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
