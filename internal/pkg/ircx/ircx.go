// Package ircx contains helpful IRC library extensions.
package ircx

import (
	"github.com/jakebailey/irc"
)

// PrivMsg creates a PRIVMSG to the target with the supplied message. The
// target will not be normalized.
func PrivMsg(target, message string) *irc.Message {
	return &irc.Message{
		Command:  "PRIVMSG",
		Params:   []string{target},
		Trailing: message,
	}
}
