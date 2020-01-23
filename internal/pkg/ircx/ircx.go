// Package ircx contains helpful IRC library extensions.
package ircx

import (
	"strings"

	"github.com/jakebailey/irc"
)

// Join creates a JOIN message using the specified channels. The channel names
// will not be normalized.
func Join(channels ...string) *irc.Message {
	return &irc.Message{
		Command: "JOIN",
		Params:  channels,
	}
}

// Part creates a PART message using the specified channels. The channel names
// will not be normalized.
func Part(channels ...string) *irc.Message {
	return &irc.Message{
		Command: "PART",
		Params:  channels,
	}
}

// PrivMsg creates a PRIVMSG to the target with the supplied message. The
// target will not be normalized.
func PrivMsg(target, message string) *irc.Message {
	return &irc.Message{
		Command:  "PRIVMSG",
		Params:   []string{target},
		Trailing: message,
	}
}

// Pass creates a PASS message with the specified pass.
func Pass(pass string) *irc.Message {
	return &irc.Message{
		Command: "PASS",
		Params:  []string{pass},
	}
}

// Nick creates a NICK message with the specified nick.
func Nick(nick string) *irc.Message {
	return &irc.Message{
		Command: "NICK",
		Params:  []string{nick},
	}
}

// CapReq creates a CAP REQ message for the specified capabilities.
func CapReq(capabilities ...string) *irc.Message {
	return &irc.Message{
		Command:  "CAP",
		Params:   []string{"REQ"},
		Trailing: strings.Join(capabilities, " "),
	}
}

// Quit returns a QUIT message.
func Quit() *irc.Message {
	return &irc.Message{Command: "QUIT"}
}

// NormalizeChannel normalizes a channel name by lowercasing the name and
// ensuring that it begins with a hash. If an empty string or a string
// containing only a hash is given, then the empty string will be returned.
func NormalizeChannel(name string) string {
	if name == "" || name == "#" {
		return ""
	}

	name = strings.ToLower(name)

	if name[0] == '#' {
		return name
	}

	return "#" + name
}

// NormalizeChannels normalizes all specified channels with NormalizeChannel,
// and returns a new slice with the normalized names.
func NormalizeChannels(names ...string) []string {
	if len(names) == 0 {
		return nil
	}

	out := make([]string, len(names))

	for i, n := range names {
		out[i] = NormalizeChannel(n)
	}

	return out
}

// Clone clones an IRC message. The new message will not have any references
// to the old message.
func Clone(m *irc.Message) *irc.Message {
	if m == nil {
		return nil
	}

	m2 := *m

	if m.Tags != nil {
		tags := make(map[string]string, len(m.Tags))
		for k, v := range m.Tags {
			tags[k] = v
		}
		m2.Tags = tags
	}

	if m.Params != nil {
		params := make([]string, len(m.Params))
		copy(params, m.Params)
		m2.Params = params
	}

	return &m2
}
