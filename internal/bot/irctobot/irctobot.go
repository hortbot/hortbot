package irctobot

import (
	"strconv"
	"strings"
	"time"

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

func (m *ircMessage) ID() string { return m.m.Tags["id"] }

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

func (m *ircMessage) parseID(tag string) int64 {
	idStr := m.m.Tags[tag]
	if idStr == "" {
		panic("irctobot: no " + tag)
	}
	id, err := strconv.ParseInt(m.m.Tags[tag], 10, 64)
	if err != nil {
		panic("irctobot: invalid " + tag)
	}
	return id
}

func (m *ircMessage) BroadcasterID() int64 {
	return m.parseID("room-id")
}

func (m *ircMessage) UserLogin() string { return strings.ToLower(m.m.Prefix.Name) }

func (m *ircMessage) UserDisplay() string {
	if displayName := m.m.Tags["display-name"]; displayName != "" {
		return displayName
	}
	return m.UserLogin()
}

func (m *ircMessage) UserID() int64 {
	return m.parseID("user-id")
}

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

func (m *ircMessage) EmoteCount() int {
	return strings.Count(m.m.Tags["emotes"], "-")
}

func (m *ircMessage) Timestamp() time.Time {
	tmiSentStr := m.m.Tags["tmi-sent-ts"]
	if tmiSentStr != "" {
		tmiSent, _ := strconv.ParseInt(tmiSentStr, 10, 64)
		return time.Unix(tmiSent/1000, 0)
	}
	return time.Time{}
}
