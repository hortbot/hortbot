package bot

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/hortbot/hortbot/internal/db/models"
	"github.com/jakebailey/irc"
)

type Session struct {
	Origin string
	M      *irc.Message

	ID         string
	RoomID     int64
	IRCChannel string
	Message    string
	Me         bool

	User        string
	UserDisplay string
	UserID      int64
	UserLevel   AccessLevel

	Bot    *Bot
	Tx     *sql.Tx
	Sender MessageSender

	Channel *models.Channel

	CommandParams string
}

func (s *Session) formatResponse(response string) string {
	response = strings.TrimSpace(response)

	if len(response) >= 4 {
		switch response[:4] {
		case "/me ", ".me ":
			return response
		}
	}

	bullet := s.Bot.bullet

	if s.Channel != nil && s.Channel.Bullet.Valid {
		bullet = s.Channel.Bullet.String
	}

	return bullet + " " + response
}

func (s *Session) Reply(response string) error {
	return s.Sender.SendMessage(s.Origin, "#"+s.IRCChannel, s.formatResponse(response))
}

func (s *Session) Replyf(format string, args ...interface{}) error {
	response := fmt.Sprintf(format, args...)
	return s.Reply(response)
}

func (s *Session) ReplyUsage(usage string) error {
	return s.Replyf("usage: %s%s", s.Channel.Prefix, usage)
}

func (s *Session) parseUserLevel() AccessLevel {
	// TODO: admin list

	if s.User == s.IRCChannel {
		return LevelBroadcaster
	}

	// Tags are present, safe to not check for nil

	tags := s.M.Tags

	if isTesting && tags["testing-admin"] != "" {
		return LevelAdmin
	}

	if tags["mod"] == "1" {
		return LevelModerator
	}

	badges := parseBadges(tags["badges"])

	switch {
	case badges["broadcaster"] != "":
		return LevelBroadcaster
	case badges["moderator"] != "":
		return LevelModerator
	case badges["subscriber"] != "", badges["vip"] != "", tags["subscriber"] == "1":
		return LevelSubscriber
	}

	if tags["user-type"] == "mod" {
		return LevelModerator
	}

	for _, owner := range s.Channel.CustomOwners {
		if s.User == owner {
			return LevelBroadcaster
		}
	}

	for _, mod := range s.Channel.CustomMods {
		if s.User == mod {
			return LevelModerator
		}
	}

	for _, reg := range s.Channel.CustomRegulars {
		if s.User == reg {
			return LevelSubscriber
		}
	}

	return LevelEveryone
}
