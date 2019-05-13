package bot

import (
	"strconv"

	"github.com/jakebailey/irc"
)

type Message struct {
	M *irc.Message
}

func NewMessage(m *irc.Message) *Message {
	return &Message{
		M: m,
	}
}

func (m *Message) Command() string {
	return m.M.Command
}

func (m *Message) Tag(key string) (value string, ok bool) {
	if len(m.M.Tags) == 0 {
		return "", false
	}

	value, ok = m.M.Tags[key]
	return value, ok
}

func (m *Message) ID() string {
	tag, _ := m.Tag("id")
	return tag
}

func (m *Message) RoomID() int64 {
	tag, _ := m.Tag("room-id")
	id, _ := strconv.ParseInt(tag, 10, 64)
	return id
}

func (m *Message) UserID() int64 {
	tag, _ := m.Tag("user-id")
	id, _ := strconv.ParseInt(tag, 10, 64)
	return id
}

func (m *Message) ChannelName() string {
	if m.Command() != "PRIVMSG" {
		return ""
	}

	params := m.M.Params
	if len(params) == 0 {
		return ""
	}

	name := params[0]

	if name == "" {
		return ""
	}

	if name[0] != '#' {
		return ""
	}

	return name[1:]
}

func (m *Message) Message() string {
	switch m.Command() {
	case "PRIVMSG":
		return m.M.Trailing
	}
	return ""
}
