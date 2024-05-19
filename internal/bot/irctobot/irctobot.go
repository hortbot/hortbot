package irctobot

import (
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/hortbot/hortbot/internal/bot"
	"github.com/hortbot/hortbot/internal/pkg/stringsx"
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

func (m *ircMessage) UserAccessLevel() bot.AccessLevel {
	tags := m.m.Tags

	if testing.Testing() {
		switch {
		case tags["testing-super-admin"] != "":
			return bot.AccessLevelSuperAdmin
		case tags["testing-admin"] != "":
			return bot.AccessLevelAdmin
		}
	}

	if m.BroadcasterID() == m.UserID() {
		return bot.AccessLevelBroadcaster
	}

	if tags["mod"] == "1" {
		return bot.AccessLevelModerator
	}

	badges := parseBadges(tags["badges"])

	switch {
	case badges["broadcaster"] != "":
		return bot.AccessLevelBroadcaster
	case badges["moderator"] != "":
		return bot.AccessLevelModerator
	case badges["vip"] != "":
		return bot.AccessLevelVIP
	case badges["subscriber"] != "", tags["subscriber"] == "1", badges["founder"] != "":
		return bot.AccessLevelSubscriber
	}

	if tags["user-type"] == "mod" {
		return bot.AccessLevelModerator
	}

	return bot.AccessLevelUnknown
}

func parseBadges(badgeTag string) map[string]string {
	badges := strings.FieldsFunc(badgeTag, func(r rune) bool { return r == ',' })

	d := make(map[string]string, len(badges))

	for _, badge := range badges {
		k, v := stringsx.SplitByte(badge, '/')
		d[k] = v
	}

	return d
}
